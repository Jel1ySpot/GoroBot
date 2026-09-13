package qbot

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"
	"time"

	"github.com/skip2/go-qrcode"
)

const (
	BindTaskCreateURL = "https://q.qq.com/lite/create_bind_task"
	BindResultPollURL = "https://q.qq.com/lite/poll_bind_result"

	BindStatusNone      = 0
	BindStatusPending   = 1
	BindStatusCompleted = 2
	BindStatusExpired   = 3
)

// QRCredentials 扫码绑定的机器人凭据
type QRCredentials struct {
	AppID      string
	Secret     string
	UserOpenID string
}

type bindTaskReq struct {
	Key string `json:"key"`
}

type bindTaskResp struct {
	Retcode int    `json:"retcode"`
	Msg     string `json:"msg"`
	Data    struct {
		TaskID string `json:"task_id"`
	} `json:"data"`
}

type pollResultReq struct {
	TaskID string `json:"task_id"`
}

type pollResultResp struct {
	Retcode int    `json:"retcode"`
	Msg     string `json:"msg"`
	Data    struct {
		Status           int    `json:"status"`
		BotAppID         string `json:"bot_appid"`
		BotEncryptSecret string `json:"bot_encrypt_secret"`
		UserOpenID       string `json:"user_openid"`
	} `json:"data"`
}

// QRConnect 发起扫码绑定流程，返回获取到的凭据
func (s *Service) QRConnect(ctx context.Context) (*QRCredentials, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 1. 生成 32 字节随机 AES Key
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("generate random aes key failed: %w", err)
		}
		keyBase64 := base64.StdEncoding.EncodeToString(key)

		// 2. 创建绑定任务
		taskID, err := createBindTask(ctx, client, keyBase64)
		if err != nil {
			s.logWarning("QBot 创建扫码绑定任务失败: %v，3秒后重试...", err)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(3 * time.Second):
				continue
			}
		}

		// 3. 构建并展示二维码
		connectURL := fmt.Sprintf(
			"https://q.qq.com/qqbot/openclaw/connect.html?task_id=%s&source=%s&_wv=2",
			urlpkg.QueryEscape(taskID),
			urlpkg.QueryEscape("GoroBot"),
		)
		s.displayQRCode(connectURL)

		// 4. 轮询扫码结果
		creds, expired, err := s.pollUntilCompleted(ctx, client, taskID, key)
		if err != nil {
			return nil, err
		}
		if expired {
			s.logWarning("QBot 扫码二维码已过期，正在刷新...")
			continue
		}

		return creds, nil
	}
}

func (s *Service) displayQRCode(connectURL string) {
	qr, err := qrcode.New(connectURL, qrcode.Medium)
	fmt.Println("\n========================= QBot 扫码登录 =========================")
	if err == nil {
		fmt.Print(qr.ToSmallString(false))
	}
	fmt.Println("请使用【手机 QQ】扫描上方二维码完成机器人绑定授权。")
	fmt.Printf("若终端无法显示二维码，可在浏览器访问以下链接生成二维码图片：\nhttps://api.qrserver.com/v1/create-qr-code/?data=%s\n", urlpkg.QueryEscape(connectURL))
	fmt.Printf("或直接在手机 QQ 打开授权链接：\n%s\n", connectURL)
	fmt.Println("=================================================================")
}

func createBindTask(ctx context.Context, client *http.Client, keyBase64 string) (string, error) {
	bodyBytes, err := json.Marshal(bindTaskReq{Key: keyBase64})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, BindTaskCreateURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoroBot-QBot/2.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var res bindTaskResp
	if err := json.Unmarshal(respBytes, &res); err != nil {
		return "", fmt.Errorf("parse create task response failed: %w", err)
	}

	if res.Retcode != 0 || res.Data.TaskID == "" {
		return "", fmt.Errorf("create bind task failed (code %d: %s)", res.Retcode, res.Msg)
	}

	return res.Data.TaskID, nil
}

func (s *Service) pollUntilCompleted(ctx context.Context, client *http.Client, taskID string, key []byte) (*QRCredentials, bool, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		case <-ticker.C:
			res, err := pollBindResult(ctx, client, taskID)
			if err != nil {
				s.logDebug("QBot poll bind result error: %v", err)
				continue
			}

			switch res.Data.Status {
			case BindStatusCompleted:
				secret, err := decryptSecret(res.Data.BotEncryptSecret, key)
				if err != nil {
					return nil, false, fmt.Errorf("decrypt bot secret failed: %w", err)
				}
				return &QRCredentials{
					AppID:      res.Data.BotAppID,
					Secret:     secret,
					UserOpenID: res.Data.UserOpenID,
				}, false, nil

			case BindStatusExpired:
				return nil, true, nil

			default:
				// PENDING 或 NONE，继续等待
			}
		}
	}
}

func pollBindResult(ctx context.Context, client *http.Client, taskID string) (*pollResultResp, error) {
	bodyBytes, err := json.Marshal(pollResultReq{TaskID: taskID})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, BindResultPollURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoroBot-QBot/2.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res pollResultResp
	if err := json.Unmarshal(respBytes, &res); err != nil {
		return nil, err
	}

	if res.Retcode != 0 {
		return nil, fmt.Errorf("poll bind result failed (code %d: %s)", res.Retcode, res.Msg)
	}

	return &res, nil
}

// decryptSecret 使用 AES-256-GCM 解密密钥
func decryptSecret(botEncryptSecretBase64 string, key []byte) (string, error) {
	cipherData, err := base64.StdEncoding.DecodeString(botEncryptSecretBase64)
	if err != nil {
		return "", fmt.Errorf("base64 decode error: %w", err)
	}

	// cipherData: nonce (前12字节) + ciphertext + tag (最后16字节)
	if len(cipherData) < 12+16 {
		return "", fmt.Errorf("encrypted data too short (length %d)", len(cipherData))
	}

	nonce := cipherData[:12]
	ciphertextAndTag := cipherData[12:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("new aes cipher error: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm error: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertextAndTag, nil)
	if err != nil {
		return "", fmt.Errorf("gcm open decrypt error: %w", err)
	}

	return string(plaintext), nil
}
