package qbot

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
)

func TestEd25519SignatureAndValidation(t *testing.T) {
	secret := "my_test_secret_12345"

	// 1. 测试种子生成与密钥对
	seed := deriveSeed(secret)
	if len(seed) != 32 {
		t.Fatalf("expected 32 bytes seed, got %d", len(seed))
	}

	priv, pub := getEd25519KeyPair(secret)
	if len(priv) != 64 || len(pub) != 32 {
		t.Fatalf("invalid key pair lengths")
	}

	// 2. 测试回调验证 (op: 13)
	plainToken := "token_abc123"
	eventTs := "1700000000"
	resp := signValidationResponse(plainToken, eventTs, secret)
	if resp.PlainToken != plainToken {
		t.Errorf("expected plainToken %s, got %s", plainToken, resp.PlainToken)
	}

	sigBytes, err := hex.DecodeString(resp.Signature)
	if err != nil {
		t.Fatalf("failed to decode hex signature: %v", err)
	}
	expectedMsg := []byte(eventTs + plainToken)
	if !ed25519.Verify(pub, expectedMsg, sigBytes) {
		t.Errorf("validation signature verification failed")
	}

	// 3. 测试 Webhook 消息签名验证
	body := []byte(`{"op":0,"t":"C2C_MESSAGE_CREATE","d":{}}`)
	timestamp := "1700000005"
	msgToSign := append([]byte(timestamp), body...)
	validSig := hex.EncodeToString(ed25519.Sign(priv, msgToSign))

	if !verifyWebhookSignature(body, timestamp, validSig, secret) {
		t.Errorf("webhook signature verification failed on valid signature")
	}

	if verifyWebhookSignature(body, timestamp, "bad_sig", secret) {
		t.Errorf("webhook signature verification should fail on invalid signature")
	}
}

func TestWebhookHandler(t *testing.T) {
	service := Create()
	service.config.Credentials.Secret = "test_secret_key"

	// 1. 测试 op: 13 验证
	valBody, _ := json.Marshal(WSPayload{
		Op: OpValidation,
		D:  json.RawMessage(`{"plain_token":"test_token","event_ts":"123456"}`),
	})
	req := httptest.NewRequest(http.MethodPost, "/bot", strings.NewReader(string(valBody)))
	rec := httptest.NewRecorder()

	service.WebhookHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for op 13, got %d", rec.Code)
	}

	var valResp ValidationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &valResp); err != nil {
		t.Fatalf("failed to parse op 13 response: %v", err)
	}
	if valResp.PlainToken != "test_token" || valResp.Signature == "" {
		t.Errorf("unexpected validation response: %+v", valResp)
	}

	// 2. 测试缺少签名的普通回调请求
	op0Body, _ := json.Marshal(WSPayload{
		Op: OpDispatch,
		T:  "C2C_MESSAGE_CREATE",
		D:  json.RawMessage(`{}`),
	})
	req2 := httptest.NewRequest(http.MethodPost, "/bot", strings.NewReader(string(op0Body)))
	rec2 := httptest.NewRecorder()
	service.WebhookHandler(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing signature headers, got %d", rec2.Code)
	}

	// 3. 测试带有效签名的回调请求
	req3 := httptest.NewRequest(http.MethodPost, "/bot", strings.NewReader(string(op0Body)))
	timestamp := "1700000010"
	priv, _ := getEd25519KeyPair("test_secret_key")
	sig := hex.EncodeToString(ed25519.Sign(priv, append([]byte(timestamp), op0Body...)))
	req3.Header.Set("X-Signature-Timestamp", timestamp)
	req3.Header.Set("X-Signature-Ed25519", sig)
	rec3 := httptest.NewRecorder()
	service.WebhookHandler(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Errorf("expected 200 on valid signature, got %d", rec3.Code)
	}
	var ack WebhookAck
	if err := json.Unmarshal(rec3.Body.Bytes(), &ack); err != nil || ack.Op != OpHttpCallbackAck {
		t.Errorf("expected op 12 ack, got: %+v", ack)
	}
}

func TestFormatAndParseID(t *testing.T) {
	id := FormatID("user", "123456")
	if id != "qbot:user&123456" {
		t.Errorf("unexpected id: %s", id)
	}

	parsed, ok := ParseID(id)
	if !ok || parsed != "123456" {
		t.Errorf("ParseID failed: got %s, ok: %v", parsed, ok)
	}

	_, badOk := ParseID("invalid:id")
	if badOk {
		t.Errorf("expected ParseID to fail for invalid id")
	}
}

func TestMessageBuilder(t *testing.T) {
	service := Create()
	msgCtx := NewMessageContext(service, nil, &Message{
		ID:      "test_msg_id",
		Content: "hello",
		Author: &User{
			ID:       "user_1",
			Username: "Tester",
		},
	})

	builder := NewMessageBuilder(msgCtx)
	builder.Text("world")
	builder.Mention(FormatID("user", "12345"))
	builder.Emoji(1)

	msgToCreate := builder.Build()
	if !strings.Contains(msgToCreate.Content, "world") {
		t.Errorf("expected content to contain world, got: %s", msgToCreate.Content)
	}
	if !strings.Contains(msgToCreate.Content, `<qqbot-at-user id="12345" />`) {
		t.Errorf("expected mention tag, got: %s", msgToCreate.Content)
	}
	if !strings.Contains(msgToCreate.Content, `<emoji:1>`) {
		t.Errorf("expected emoji tag, got: %s", msgToCreate.Content)
	}
}

func TestTokenManager(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokenResp{
			AccessToken: "mock_token_12345",
			ExpiresIn:   json.Number("7200"),
		})
	}))
	defer server.Close()

	mgr := NewTokenManager(server.URL, nil)

	// 第一次调用，向服务端请求
	token1, err := mgr.GetAccessToken(context.Background(), "app_1", "sec_1")
	if err != nil || token1 != "mock_token_12345" {
		t.Fatalf("first GetAccessToken failed: %v, %s", err, token1)
	}
	if callCount != 1 {
		t.Errorf("expected callCount 1, got %d", callCount)
	}

	// 第二次调用，应命中缓存
	token2, err := mgr.GetAccessToken(context.Background(), "app_1", "sec_1")
	if err != nil || token2 != "mock_token_12345" {
		t.Fatalf("second GetAccessToken failed: %v, %s", err, token2)
	}
	if callCount != 1 {
		t.Errorf("expected callCount still 1 (cached), got %d", callCount)
	}

	// 清理缓存后再调用，重新请求
	mgr.ClearCache("app_1")
	token3, err := mgr.GetAccessToken(context.Background(), "app_1", "sec_1")
	if err != nil || token3 != "mock_token_12345" {
		t.Fatalf("third GetAccessToken failed: %v, %s", err, token3)
	}
	if callCount != 2 {
		t.Errorf("expected callCount 2 after clear cache, got %d", callCount)
	}
}

func TestClientAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		auth := r.Header.Get("Authorization")
		if auth != "QQBot dummy_token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/users/@me":
			_ = json.NewEncoder(w).Encode(User{
				ID:       "10001",
				Username: "TestBot",
			})
		case "/v2/users/test_user/messages":
			_ = json.NewEncoder(w).Encode(MessageResponse{
				ID: "msg_sent_1",
			})
		case "/v2/groups/test_group/messages":
			_ = json.NewEncoder(w).Encode(MessageResponse{
				ID: "msg_sent_2",
			})
		case "/gateway":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"url": "wss://gateway.mock",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	mgr := NewTokenManager("", nil)
	mgr.tokens["test_app"] = &CachedToken{
		Token:     "dummy_token",
		ExpiresAt: time.Now().Add(time.Hour),
	}

	client := NewClient(server.URL, Credentials{AppID: "test_app", Secret: "dummy_secret"}, mgr, nil, false)

	// 测试 Me
	me, err := client.Me(context.Background())
	if err != nil || me.ID != "10001" || me.Username != "TestBot" {
		t.Fatalf("Me() failed: %v, %+v", err, me)
	}

	// 测试 PostC2CMessage
	c2cMsg, err := client.PostC2CMessage(context.Background(), "test_user", &MessageToCreate{Content: "ping"})
	if err != nil || c2cMsg.ID != "msg_sent_1" {
		t.Fatalf("PostC2CMessage() failed: %v, %+v", err, c2cMsg)
	}

	// 测试 PostGroupMessage
	grpMsg, err := client.PostGroupMessage(context.Background(), "test_group", &MessageToCreate{Content: "group ping"})
	if err != nil || grpMsg.ID != "msg_sent_2" {
		t.Fatalf("PostGroupMessage() failed: %v, %+v", err, grpMsg)
	}

	// 测试 GetGatewayURL
	gwURL, err := client.GetGatewayURL(context.Background())
	if err != nil || gwURL != "wss://gateway.mock" {
		t.Fatalf("GetGatewayURL() failed: %v, %s", err, gwURL)
	}
}

func TestDecryptSecret(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)

	plainSecret := "actual_app_secret_99887766"

	// 使用 AES-256-GCM 加密
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("aes.NewCipher failed: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("cipher.NewGCM failed: %v", err)
	}

	nonce := make([]byte, 12)
	_, _ = rand.Read(nonce)

	ciphertext := gcm.Seal(nil, nonce, []byte(plainSecret), nil)
	// payload: nonce (12) + ciphertext (includes 16-byte tag at the end)
	fullPayload := append(nonce, ciphertext...)
	encodedPayload := base64.StdEncoding.EncodeToString(fullPayload)

	// 测试解密
	decrypted, err := decryptSecret(encodedPayload, key)
	if err != nil {
		t.Fatalf("decryptSecret failed: %v", err)
	}
	if decrypted != plainSecret {
		t.Errorf("expected decrypted secret %s, got %s", plainSecret, decrypted)
	}
}

func TestIsConfigured(t *testing.T) {
	cfg1 := Config{
		Credentials: Credentials{AppID: "", Secret: ""},
	}
	if cfg1.IsConfigured() {
		t.Errorf("empty credentials should not be configured")
	}

	cfg2 := Config{
		Credentials: Credentials{AppID: "your app id", Secret: "your secret key"},
	}
	if cfg2.IsConfigured() {
		t.Errorf("placeholder credentials should not be configured")
	}

	cfg3 := Config{
		Credentials: Credentials{AppID: "10200300", Secret: "actual_secret"},
	}
	if !cfg3.IsConfigured() {
		t.Errorf("valid credentials should be configured")
	}
}

func TestFeaturesAndKeyboard(t *testing.T) {
	service := Create()
	if !service.SupportsFeature(botc.FeatureInlineKeyboard) {
		t.Errorf("expected qbot to support FeatureInlineKeyboard")
	}
	if !service.SupportsFeature(botc.FeatureMarkdown) {
		t.Errorf("expected qbot to support FeatureMarkdown")
	}

	msgCtx := NewMessageContext(service, nil, &Message{
		ID:      "test_msg_id",
		Content: "hello",
		Author: &User{
			ID:       "user_1",
			Username: "Tester",
		},
	})
	if !msgCtx.SupportsFeature(botc.FeatureInlineKeyboard) {
		t.Errorf("expected msgCtx to support FeatureInlineKeyboard")
	}

	kb := botc.NewInlineKeyboard().AddRow(
		botc.NewURLButton("官网", "https://bot.q.qq.com"),
		botc.NewCommandButton("签到", "/sign", true),
	)

	builder := NewMessageBuilder(msgCtx)
	builder.InlineKeyboard(kb)
	msg := builder.Build()

	if msg.Keyboard == nil || msg.Keyboard.Content == nil {
		t.Fatalf("expected keyboard content to be populated")
	}
	if len(msg.Keyboard.Content.Rows) != 1 || len(msg.Keyboard.Content.Rows[0].Buttons) != 2 {
		t.Fatalf("unexpected keyboard rows: %+v", msg.Keyboard.Content.Rows)
	}
	btn1 := msg.Keyboard.Content.Rows[0].Buttons[0]
	btn2 := msg.Keyboard.Content.Rows[0].Buttons[1]
	if btn1.Action.Type != 0 || btn1.Action.Data != "https://bot.q.qq.com" {
		t.Errorf("unexpected btn1 action: %+v", btn1.Action)
	}
	if btn2.Action.Type != 2 || btn2.Action.Data != "/sign" || !btn2.Action.Enter {
		t.Errorf("unexpected btn2 action: %+v", btn2.Action)
	}
}

func TestTokenManager_FetchToken(t *testing.T) {
	// 测试 QQ 开放平台返回 expires_in 为字符串及数字的不同情况
	tests := []struct {
		name       string
		respJSON   string
		wantExpire int
		wantErr    bool
	}{
		{
			name:       "expires_in as string",
			respJSON:   `{"access_token":"token_str","expires_in":"7200"}`,
			wantExpire: 7200,
			wantErr:    false,
		},
		{
			name:       "expires_in as integer",
			respJSON:   `{"access_token":"token_int","expires_in":7200}`,
			wantExpire: 7200,
			wantErr:    false,
		},
		{
			name:       "missing access_token",
			respJSON:   `{"code":-1,"msg":"invalid secret"}`,
			wantExpire: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.respJSON))
			}))
			defer server.Close()

			mgr := NewTokenManager(server.URL, nil)
			token, err := mgr.GetAccessToken(context.Background(), "appid", "secret")
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetAccessToken error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && token == "" {
				t.Fatalf("expected non-empty token")
			}
		})
	}
}

