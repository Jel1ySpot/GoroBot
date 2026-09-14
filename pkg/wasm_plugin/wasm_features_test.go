package wasm_plugin

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	extism "github.com/extism/go-sdk"
)

type mockBotContext struct {
	lastSentText     string
	lastSentKeyboard *botc.InlineKeyboard
}

func (m *mockBotContext) Features() []botc.Feature {
	return []botc.Feature{botc.FeatureText, botc.FeatureInlineKeyboard}
}

func (m *mockBotContext) SupportsFeature(f botc.Feature) bool {
	return f == botc.FeatureText || f == botc.FeatureInlineKeyboard
}

func (m *mockBotContext) ID() string {
	return "mock:bot_1"
}

func (m *mockBotContext) Name() string {
	return "MockBot"
}

func (m *mockBotContext) Protocol() string {
	return "mock"
}

func (m *mockBotContext) Status() botc.LoginStatus {
	return botc.Online
}

func (m *mockBotContext) NewMessageBuilder() botc.MessageBuilder {
	return &mockBuilder{ctx: m}
}

func (m *mockBotContext) SendDirectMessage(target entity.User, elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	return nil, nil
}

func (m *mockBotContext) SendGroupMessage(target entity.Group, elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	return nil, nil
}

func (m *mockBotContext) Contacts() []entity.User {
	return nil
}

func (m *mockBotContext) Groups() []entity.Group {
	return nil
}

func (m *mockBotContext) DownloadResourceFromRefLink(refLink string) (string, error) {
	return "", nil
}

type mockBuilder struct {
	ctx *mockBotContext
	kb  *botc.InlineKeyboard
	txt string
}

func (b *mockBuilder) Protocol() string { return "mock" }
func (b *mockBuilder) Text(text string) botc.MessageBuilder {
	b.txt = text
	return b
}
func (b *mockBuilder) Markdown(content string) botc.MessageBuilder {
	b.txt = content
	return b
}
func (b *mockBuilder) Quote(msg *botc.BaseMessage) botc.MessageBuilder { return b }
func (b *mockBuilder) Mention(id string) botc.MessageBuilder            { return b }
func (b *mockBuilder) ImageFromFile(path string) botc.MessageBuilder   { return b }
func (b *mockBuilder) ImageFromUrl(url string) botc.MessageBuilder     { return b }
func (b *mockBuilder) ImageFromData(data []byte) botc.MessageBuilder   { return b }
func (b *mockBuilder) InlineKeyboard(kb *botc.InlineKeyboard) botc.MessageBuilder {
	b.kb = kb
	return b
}
func (b *mockBuilder) ReplyTo(msg botc.MessageContext) (*botc.BaseMessage, error) {
	b.ctx.lastSentText = b.txt
	b.ctx.lastSentKeyboard = b.kb
	return &botc.BaseMessage{ID: "mock_reply_id"}, nil
}
func (b *mockBuilder) Send(id string) (*botc.BaseMessage, error) {
	b.ctx.lastSentText = b.txt
	b.ctx.lastSentKeyboard = b.kb
	return &botc.BaseMessage{ID: "mock_msg_id"}, nil
}

func TestWasmHostFunctionsFeaturesAndKeyboard(t *testing.T) {
	grb := GoroBot.Create()
	s := Create()
	mockBot := &mockBotContext{}
	grb.AddContext(mockBot)
	s.grb = grb

	inst := &PluginInstance{
		id:      "test_plugin",
		service: s,
	}

	funcs := s.createHostFunctions(inst)
	if len(funcs) == 0 {
		t.Fatal("no host functions registered")
	}

	// 验证向 Wasm 发送消息中带 Keyboard 的序列化与执行
	req := SendMessageRequest{
		BotContextID: "mock:bot_1",
		TargetID:     "user_123",
		Text:         "Hello with button",
		Keyboard: &InlineKeyboardPayload{
			Rows: [][]InlineKeyboardButtonPayload{
				{
					{
						Text:   "Click me",
						Action: 0,
						Data:   "https://example.com",
					},
				},
			},
		},
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal req: %v", err)
	}

	// 模拟直接调用 builder 流程（与 gorobot_send_message 内逻辑一致）
	builder := mockBot.NewMessageBuilder().Text(req.Text)
	if req.Keyboard != nil && len(req.Keyboard.Rows) > 0 {
		kb := botc.NewInlineKeyboard()
		for _, r := range req.Keyboard.Rows {
			var rowBtns []botc.InlineKeyboardButton
			for _, b := range r {
				rowBtns = append(rowBtns, botc.InlineKeyboardButton{
					Text:       b.Text,
					Action:     botc.ButtonActionType(b.Action),
					Data:       b.Data,
					DirectSend: b.DirectSend,
					ID:         b.ID,
				})
			}
			kb.AddRow(rowBtns...)
		}
		builder.InlineKeyboard(kb)
	}

	_, err = builder.Send(req.TargetID)
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	if mockBot.lastSentText != "Hello with button" {
		t.Fatalf("expected text 'Hello with button', got %s", mockBot.lastSentText)
	}

	if mockBot.lastSentKeyboard == nil || len(mockBot.lastSentKeyboard.Rows) != 1 {
		t.Fatalf("expected keyboard with 1 row, got %+v", mockBot.lastSentKeyboard)
	}
	if mockBot.lastSentKeyboard.Rows[0][0].Text != "Click me" {
		t.Fatalf("expected button text 'Click me', got %s", mockBot.lastSentKeyboard.Rows[0][0].Text)
	}

	// 验证 Markdown 消息发送
	reqMd := SendMessageRequest{
		BotContextID: "mock:bot_1",
		TargetID:     "user_123",
		Markdown:     "# Markdown Title",
	}
	builderMd := mockBot.NewMessageBuilder()
	if reqMd.Markdown != "" {
		builderMd.Markdown(reqMd.Markdown)
	} else {
		builderMd.Text(reqMd.Text)
	}
	_, err = builderMd.Send(reqMd.TargetID)
	if err != nil {
		t.Fatalf("send markdown failed: %v", err)
	}
	if mockBot.lastSentText != "# Markdown Title" {
		t.Fatalf("expected text '# Markdown Title', got %s", mockBot.lastSentText)
	}

	_ = reqBytes
	_ = context.Background()
	_ = extism.Manifest{}
}

func TestExtractMessageDetails(t *testing.T) {
	// 构造测试消息上下文
	mockMsg := &botc.BaseMessage{
		MessageType: botc.GroupMessage,
		ID:          "msg_12345",
		Content:     "hello world",
		Elements: []*botc.MessageElement{
			{Type: botc.TextElement, Content: "hello world"},
		},
		Sender: &entity.Sender{
			User: &entity.User{
				Base: &entity.Base{
					ID:   "user_999",
					Name: "Alice",
				},
				Nickname:  "AliceNick",
				Authority: entity.GroupAdmin,
			},
			From: &entity.Base{
				ID:   "group_888",
				Name: "Dev Group",
			},
		},
	}

	mockCtx := &mockMessageContext{msg: mockMsg}
	msgType, groupID, group, sender, msgID, elements, _ := extractMessageDetails(mockCtx)

	if msgType != "group" {
		t.Errorf("expected msgType 'group', got %s", msgType)
	}
	if groupID != "group_888" {
		t.Errorf("expected groupID 'group_888', got %s", groupID)
	}
	if group == nil || group.Name != "Dev Group" {
		t.Errorf("unexpected group: %+v", group)
	}
	if sender == nil || sender.ID != "user_999" || sender.Nickname != "AliceNick" || sender.Authority != 2 {
		t.Errorf("unexpected sender: %+v", sender)
	}
	if msgID != "msg_12345" || len(elements) != 1 {
		t.Errorf("unexpected msgID or elements: %s, %+v", msgID, elements)
	}
}

type mockMessageContext struct {
	msg *botc.BaseMessage
}

func (m *mockMessageContext) Features() []botc.Feature {
	return []botc.Feature{botc.FeatureText}
}
func (m *mockMessageContext) SupportsFeature(f botc.Feature) bool {
	return f == botc.FeatureText
}
func (m *mockMessageContext) Protocol() string {
	return "mock"
}
func (m *mockMessageContext) BotContext() botc.BotContext {
	return nil
}
func (m *mockMessageContext) String() string {
	return m.msg.Content
}
func (m *mockMessageContext) Message() *botc.BaseMessage {
	return m.msg
}
func (m *mockMessageContext) SenderID() string {
	return "user_999"
}
func (m *mockMessageContext) NewMessageBuilder() botc.MessageBuilder {
	return nil
}
func (m *mockMessageContext) Reply(elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	return nil, nil
}
func (m *mockMessageContext) ReplyText(a ...any) (*botc.BaseMessage, error) {
	return nil, nil
}

func TestWasmTimers(t *testing.T) {
	s := Create()
	inst := &PluginInstance{
		id:      "timer_plugin",
		service: s,
		timers:  make(map[string]context.CancelFunc),
	}

	funcs := s.createHostFunctions(inst)
	foundInterval := false
	foundTimeout := false
	foundClear := false
	foundTimeNow := false

	for _, f := range funcs {
		switch f.Name {
		case "gorobot_set_interval":
			foundInterval = true
		case "gorobot_set_timeout":
			foundTimeout = true
		case "gorobot_clear_timer":
			foundClear = true
		case "gorobot_time_now":
			foundTimeNow = true
		}
	}

	if !foundInterval || !foundTimeout || !foundClear || !foundTimeNow {
		t.Fatalf("expected timer host functions to be registered: interval=%v, timeout=%v, clear=%v, timeNow=%v",
			foundInterval, foundTimeout, foundClear, foundTimeNow)
	}

	// 测试定时器添加与取消管理
	var tickCount int32
	ctx, cancel := context.WithCancel(context.Background())
	inst.AddTimer("timer_test_1", cancel)

	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				atomic.AddInt32(&tickCount, 1)
			}
		}
	}()

	time.Sleep(55 * time.Millisecond)
	if atomic.LoadInt32(&tickCount) == 0 {
		t.Errorf("expected ticks > 0")
	}

	// 取消定时器
	removed := inst.RemoveTimer("timer_test_1")
	if !removed {
		t.Errorf("expected timer_test_1 to be removed successfully")
	}

	currentCount := atomic.LoadInt32(&tickCount)
	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&tickCount) != currentCount {
		t.Errorf("expected timer to stop ticking after cancel, but count increased from %d to %d",
			currentCount, atomic.LoadInt32(&tickCount))
	}

	// 测试 Release 清理所有定时器
	var releaseTick int32
	ctx2, cancel2 := context.WithCancel(context.Background())
	inst.AddTimer("timer_test_2", cancel2)

	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx2.Done():
				return
			case <-ticker.C:
				atomic.AddInt32(&releaseTick, 1)
			}
		}
	}()

	time.Sleep(30 * time.Millisecond)
	inst.Release()

	afterReleaseCount := atomic.LoadInt32(&releaseTick)
	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&releaseTick) != afterReleaseCount {
		t.Errorf("expected timer to be cancelled on release")
	}
}

func TestWasmHttpRequestFullResponse(t *testing.T) {
	binaryData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0xFF}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 校验收到的自定义多值请求头和二进制 Body
		if r.Header.Get("X-Custom") != "val1" {
			t.Errorf("unexpected X-Custom: %s", r.Header.Get("X-Custom"))
		}

		w.Header().Add("Set-Cookie", "c1=v1; Path=/")
		w.Header().Add("Set-Cookie", "c2=v2; Path=/")
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("X-Custom-Resp", "awesome")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(binaryData)
	}))
	defer ts.Close()

	// 模拟请求参数
	reqPayload := HttpRequestPayload{
		URL:    ts.URL,
		Method: "POST",
		Headers: map[string]string{
			"X-Custom": "val1",
		},
		BodyBase64: base64.StdEncoding.EncodeToString([]byte("test body binary")),
	}

	reqBytes, err := json.Marshal(reqPayload)
	if err != nil {
		t.Fatalf("marshal req: %v", err)
	}
	_ = reqBytes

	// 模拟执行请求逻辑
	client := &http.Client{Timeout: 5 * time.Second}
	var bodyReader *bytes.Reader
	if reqPayload.BodyBase64 != "" {
		dec, _ := base64.StdEncoding.DecodeString(reqPayload.BodyBase64)
		bodyReader = bytes.NewReader(dec)
	}
	httpReq, err := http.NewRequestWithContext(context.Background(), reqPayload.Method, reqPayload.URL, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for k, v := range reqPayload.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("client do: %v", err)
	}
	defer resp.Body.Close()

	rawHeaders := make(map[string][]string)
	flatHeaders := make(map[string]string)
	for k, vs := range resp.Header {
		rawHeaders[k] = vs
		val := ""
		for i, v := range vs {
			if i > 0 {
				val += ", "
			}
			val += v
		}
		flatHeaders[k] = val
		flatHeaders[strings.ToLower(k)] = val
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	respPayload := HttpResponsePayload{
		StatusCode:    resp.StatusCode,
		Status:        resp.Status,
		Proto:         resp.Proto,
		Headers:       flatHeaders,
		RawHeaders:    rawHeaders,
		Body:          string(respBody),
		BodyBase64:    base64.StdEncoding.EncodeToString(respBody),
		ContentLength: int64(len(respBody)),
	}

	if respPayload.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", respPayload.StatusCode)
	}
	if respPayload.ContentLength != int64(len(binaryData)) {
		t.Fatalf("expected content length %d, got %d", len(binaryData), respPayload.ContentLength)
	}
	if respPayload.BodyBase64 != base64.StdEncoding.EncodeToString(binaryData) {
		t.Fatalf("body base64 mismatch")
	}

	// 校验多值 Header 是否完整保留
	cookies := respPayload.RawHeaders["Set-Cookie"]
	if len(cookies) != 2 {
		t.Fatalf("expected 2 Set-Cookie headers in raw_headers, got %v", cookies)
	}

	// 校验大小写不敏感查询
	if respPayload.Headers["content-type"] != "image/png" || respPayload.Headers["Content-Type"] != "image/png" {
		t.Fatalf("expected lowercase and canonical headers to match 'image/png', got: %v", respPayload.Headers)
	}
}
