package qbot

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
)

const (
	DefaultAPIBaseURL = "https://api.sgroup.qq.com"
)

// Client QQBot OpenAPI 客户端
type Client struct {
	baseURL      string
	tokenManager *TokenManager
	credentials  Credentials
	httpClient   *http.Client
	logger       logger.Inst
	debug        bool
}

// NewClient 创建 OpenAPI 客户端
func NewClient(baseURL string, creds Credentials, tokenMgr *TokenManager, log logger.Inst, debug bool) *Client {
	if baseURL == "" {
		baseURL = DefaultAPIBaseURL
	}
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		tokenManager: tokenMgr,
		credentials:  creds,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		logger:       log,
		debug:        debug,
	}
}

// getNextMsgSeq 生成消息序号 (1..65535)
func getNextMsgSeq() int {
	return 1 + rand.Intn(65535)
}

// Request 发起带鉴权的 HTTP 请求
func (c *Client) Request(ctx context.Context, method, path string, body any, result any) error {
	token, err := c.tokenManager.GetAccessToken(ctx, c.credentials.AppID, c.credentials.ClientSecret())
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	url := path
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = fmt.Sprintf("%s%s", c.baseURL, path)
	}

	var bodyReader io.Reader
	if body != nil {
		var reqData []byte
		if b, ok := body.([]byte); ok {
			reqData = b
		} else {
			var err error
			reqData, err = json.Marshal(body)
			if err != nil {
				return fmt.Errorf("marshal request body: %w", err)
			}
		}
		bodyReader = bytes.NewReader(reqData)

		if c.logger != nil {
			c.logger.Debug("QBot API >>> %s %s body: %s", method, url, string(reqData))
		}
	} else if c.logger != nil {
		c.logger.Debug("QBot API >>> %s %s", method, url)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("QQBot %s", token))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoroBot-QBot/2.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body failed: %w", err)
	}

	if c.logger != nil {
		c.logger.Debug("QBot API <<< [%d] %s", resp.StatusCode, string(respBytes))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("api %s %s failed (status %d): %s", method, path, resp.StatusCode, string(respBytes))
	}

	if result != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, result); err != nil {
			return fmt.Errorf("unmarshal response failed: %w", err)
		}
	}

	return nil
}

// Me 获取机器人自身信息
func (c *Client) Me(ctx context.Context) (*User, error) {
	var user User
	if err := c.Request(ctx, http.MethodGet, "/users/@me", nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// PostC2CMessage 发送单聊(私聊)消息
func (c *Client) PostC2CMessage(ctx context.Context, openid string, msg *MessageToCreate) (*Message, error) {
	if msg.MsgSeq == 0 {
		msg.MsgSeq = getNextMsgSeq()
	}
	path := fmt.Sprintf("/v2/users/%s/messages", openid)
	var resp MessageResponse
	if err := c.Request(ctx, http.MethodPost, path, msg, &resp); err != nil {
		return nil, err
	}
	return &Message{
		ID:            resp.ID,
		Content:       msg.Content,
		DirectMessage: true,
	}, nil
}

// PostGroupMessage 发送群聊消息
func (c *Client) PostGroupMessage(ctx context.Context, groupOpenid string, msg *MessageToCreate) (*Message, error) {
	if msg.MsgSeq == 0 {
		msg.MsgSeq = getNextMsgSeq()
	}
	path := fmt.Sprintf("/v2/groups/%s/messages", groupOpenid)
	var resp MessageResponse
	if err := c.Request(ctx, http.MethodPost, path, msg, &resp); err != nil {
		return nil, err
	}
	return &Message{
		ID:      resp.ID,
		Content: msg.Content,
		GroupID: groupOpenid,
	}, nil
}

// PostMessage 发送频道子频道消息
func (c *Client) PostMessage(ctx context.Context, channelID string, msg *MessageToCreate) (*Message, error) {
	path := fmt.Sprintf("/channels/%s/messages", channelID)
	var resp Message
	if err := c.Request(ctx, http.MethodPost, path, msg, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UploadFileData 上传媒体文件数据
func (c *Client) UploadFileData(ctx context.Context, scope, targetID string, fileType uint64, data []byte) (*FileInfo, error) {
	var path string
	if scope == "user" {
		path = fmt.Sprintf("/v2/users/%s/files", targetID)
	} else {
		path = fmt.Sprintf("/v2/groups/%s/files", targetID)
	}

	body := FileUpload{
		FileType:   int(fileType),
		FileData:   base64.StdEncoding.EncodeToString(data),
		SrvSendMsg: false,
	}

	var res FileInfo
	if err := c.Request(ctx, http.MethodPost, path, body, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// UploadFileURL 通过网络 URL 上传媒体文件
func (c *Client) UploadFileURL(ctx context.Context, scope, targetID string, fileType uint64, url string) (*FileInfo, error) {
	var path string
	if scope == "user" {
		path = fmt.Sprintf("/v2/users/%s/files", targetID)
	} else {
		path = fmt.Sprintf("/v2/groups/%s/files", targetID)
	}

	body := FileUpload{
		FileType:   int(fileType),
		URL:        url,
		SrvSendMsg: false,
	}

	var res FileInfo
	if err := c.Request(ctx, http.MethodPost, path, body, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetGatewayURL 获取 WebSocket 网关地址
func (c *Client) GetGatewayURL(ctx context.Context) (string, error) {
	var res struct {
		URL string `json:"url"`
	}
	if err := c.Request(ctx, http.MethodGet, "/gateway", nil, &res); err != nil {
		return "", err
	}
	return res.URL, nil
}
