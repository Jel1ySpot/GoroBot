package qbot

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// OpCodes
const (
	OpDispatch        = 0
	OpHttpCallbackAck = 12
	OpValidation      = 13
)

// deriveSeed 从机器人的 AppSecret 生成 32 字节 Ed25519 种子
func deriveSeed(secret string) []byte {
	seed := secret
	for len(seed) < 32 {
		seed += secret
	}
	return []byte(seed[:32])
}

// getEd25519KeyPair 获取 Ed25519 密钥对
func getEd25519KeyPair(secret string) (ed25519.PrivateKey, ed25519.PublicKey) {
	seed := deriveSeed(secret)
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)
	return priv, pub
}

// verifyWebhookSignature 验证回调请求的 Ed25519 签名
func verifyWebhookSignature(body []byte, timestamp, signature, secret string) bool {
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	_, pub := getEd25519KeyPair(secret)
	msg := append([]byte(timestamp), body...)
	return ed25519.Verify(pub, msg, sigBytes)
}

// signValidationResponse 计算回调验证响应 (op: 13)
func signValidationResponse(plainToken, eventTs, secret string) ValidationResponse {
	priv, _ := getEd25519KeyPair(secret)
	msg := []byte(eventTs + plainToken)
	sig := ed25519.Sign(priv, msg)
	return ValidationResponse{
		PlainToken: plainToken,
		Signature:  hex.EncodeToString(sig),
	}
}

// WebhookHandler HTTP Webhook 处理入口
func (s *Service) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Read body error", http.StatusBadRequest)
		return
	}

	var payload WSPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid json", http.StatusBadRequest)
		return
	}

	secret := s.config.Credentials.ClientSecret()

	// 1. op: 13 回调地址验证
	if payload.Op == OpValidation {
		var val ValidationPayload
		if err := json.Unmarshal(payload.D, &val); err != nil {
			http.Error(w, "Invalid validation payload", http.StatusBadRequest)
			return
		}

		s.logInfo("QBot handling webhook callback URL validation")
		respData := signValidationResponse(val.PlainToken, val.EventTs, secret)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(respData)
		return
	}

	// 2. 签名验证
	timestamp := r.Header.Get("X-Signature-Timestamp")
	signature := r.Header.Get("X-Signature-Ed25519")

	if timestamp == "" || signature == "" {
		s.logWarning("QBot webhook missing signature headers")
		http.Error(w, "Missing signature headers", http.StatusUnauthorized)
		return
	}

	if !verifyWebhookSignature(body, timestamp, signature, secret) {
		s.logWarning("QBot webhook signature verification failed")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// 3. op: 0 业务事件分发 (异步处理，立即返回 op: 12 ACK)
	if payload.Op == OpDispatch {
		go func(p WSPayload) {
			if err := s.handleWebhookDispatch(&p); err != nil {
				s.logError("QBot dispatch event failed: %v", err)
			}
		}(payload)
	}

	// 立即返回 op: 12 ACK
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(WebhookAck{
		Op: OpHttpCallbackAck,
		D:  0,
	})
}

// runHttp 启动 HTTP Webhook 服务与资源服务器
func (s *Service) runHttp() error {
	conf := s.config.GetWebhookConfig()
	if conf.Port == 0 {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc(conf.Path, s.WebhookHandler)
	mux.HandleFunc("/resource/", s.resourceService)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", conf.Host, conf.Port),
		Handler: mux,
	}

	errChan := make(chan error, 1)

	go func() {
		if conf.TLS.CertPath != "" && conf.TLS.KeyPath != "" {
			s.logInfo("QBot serving HTTPS on %s:%d", conf.Host, conf.Port)
			if err := server.ListenAndServeTLS(conf.TLS.CertPath, conf.TLS.KeyPath); err != nil && err != http.ErrServerClosed {
				errChan <- err
			}
		} else {
			s.logInfo("QBot serving HTTP on %s:%d", conf.Host, conf.Port)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				errChan <- err
			}
		}
		close(errChan)
	}()

	select {
	case err := <-errChan:
		return err
	case <-time.After(200 * time.Millisecond):
	}

	go func() {
		<-s.ctx.Done()
		_ = server.Close()
	}()

	return nil
}

func (s *Service) resourceService(writer http.ResponseWriter, request *http.Request) {
	path := strings.TrimPrefix(request.URL.Path, "/resource")
	id := strings.Trim(path, "/#")
	s.logDebug("QBot getting resource %s through api", id)
	if id == "" || strings.Contains(id, "/") {
		http.Error(writer, "Invalid resource ID", http.StatusBadRequest)
		return
	}

	resourcePath, err := s.grb.LoadResourceFromID(id)
	if err != nil {
		http.Error(writer, "Resource not found", http.StatusNotFound)
		return
	}

	data, err := os.ReadFile(resourcePath)
	if err != nil {
		http.Error(writer, "Resource not found", http.StatusNotFound)
		return
	}

	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(data)
}
