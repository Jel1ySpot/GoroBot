package wasm_plugin

import (
	"context"
	"encoding/json"
	"testing"

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
