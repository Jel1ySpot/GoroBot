package qbot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Jel1ySpot/GoroBot/pkg/core/logger"
)

const (
	DefaultTokenBaseURL = "https://bots.qq.com"
	TokenPath           = "/app/getAppAccessToken"
)

// CachedToken 缓存的 Access Token
type CachedToken struct {
	Token     string
	ExpiresAt time.Time
}

// TokenManager AccessToken 管理器，支持缓存与后台自动刷新
type TokenManager struct {
	baseURL    string
	httpClient *http.Client
	logger     logger.Inst

	mu     sync.RWMutex
	tokens map[string]*CachedToken
}

// NewTokenManager 创建 TokenManager
func NewTokenManager(baseURL string, log logger.Inst) *TokenManager {
	if baseURL == "" {
		baseURL = DefaultTokenBaseURL
	}
	return &TokenManager{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     log,
		tokens:     make(map[string]*CachedToken),
	}
}

type tokenReq struct {
	AppID        string `json:"appId"`
	ClientSecret string `json:"clientSecret"`
}

type tokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Msg         string `json:"msg,omitempty"`
	Code        int    `json:"code,omitempty"`
}

// GetAccessToken 获取 Access Token，优先读取未过期缓存
func (m *TokenManager) GetAccessToken(ctx context.Context, appID, clientSecret string) (string, error) {
	m.mu.RLock()
	cached, ok := m.tokens[appID]
	if ok && time.Now().Add(5*time.Minute).Before(cached.ExpiresAt) {
		token := cached.Token
		m.mu.RUnlock()
		return token, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查
	if cached, ok := m.tokens[appID]; ok && time.Now().Add(5*time.Minute).Before(cached.ExpiresAt) {
		return cached.Token, nil
	}

	token, expiresIn, err := m.fetchToken(ctx, appID, clientSecret)
	if err != nil {
		return "", err
	}

	m.tokens[appID] = &CachedToken{
		Token:     token,
		ExpiresAt: time.Now().Add(time.Duration(expiresIn) * time.Second),
	}

	return token, nil
}

// fetchToken 发起 HTTP POST 获取 Token
func (m *TokenManager) fetchToken(ctx context.Context, appID, clientSecret string) (string, int, error) {
	url := fmt.Sprintf("%s%s", m.baseURL, TokenPath)

	bodyBytes, err := json.Marshal(tokenReq{
		AppID:        appID,
		ClientSecret: clientSecret,
	})
	if err != nil {
		return "", 0, fmt.Errorf("marshal token request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", 0, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoroBot-QBot/2.0")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("request token failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("read token response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("token request returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var res tokenResp
	if err := json.Unmarshal(respBytes, &res); err != nil {
		return "", 0, fmt.Errorf("parse token response failed: %w", err)
	}

	if res.AccessToken == "" {
		return "", 0, fmt.Errorf("token response missing access_token (code %d: %s)", res.Code, res.Msg)
	}

	expiresIn := res.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 7200
	}

	return res.AccessToken, expiresIn, nil
}

// StartBackgroundRefresh 启动后台自动刷新协程
func (m *TokenManager) StartBackgroundRefresh(ctx context.Context, appID, clientSecret string) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			_, err := m.GetAccessToken(ctx, appID, clientSecret)
			if err != nil && m.logger != nil {
				m.logger.Warning("QBot background token refresh failed: %v", err)
			}

			// 默认每50分钟刷新一次
			select {
			case <-ctx.Done():
				return
			case <-time.After(50 * time.Minute):
			}
		}
	}()
}

// ClearCache 清理缓存
func (m *TokenManager) ClearCache(appID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tokens, appID)
}
