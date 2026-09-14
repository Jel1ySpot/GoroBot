package GoroBot

import (
	"encoding/json"
	"testing"

	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
)

type mockBotContext struct {
	id string
}

func (m *mockBotContext) Features() []botc.Feature                     { return nil }
func (m *mockBotContext) SupportsFeature(f botc.Feature) bool         { return false }
func (m *mockBotContext) ID() string                                  { return m.id }
func (m *mockBotContext) Name() string                                { return "mock" }
func (m *mockBotContext) Protocol() string                            { return "mock" }
func (m *mockBotContext) Status() botc.LoginStatus                    { return botc.Online }
func (m *mockBotContext) NewMessageBuilder() botc.MessageBuilder      { return nil }
func (m *mockBotContext) SendDirectMessage(target entity.User, elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	return nil, nil
}
func (m *mockBotContext) SendGroupMessage(target entity.Group, elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	return nil, nil
}
func (m *mockBotContext) Contacts() []entity.User                     { return nil }
func (m *mockBotContext) Groups() []entity.Group                      { return nil }
func (m *mockBotContext) DownloadResourceFromRefLink(refLink string) (string, error) {
	return "", nil
}

type mockMsgContext struct {
	botCtx   botc.BotContext
	protocol string
	senderID string
	baseMsg  *botc.BaseMessage
}

func (m *mockMsgContext) Features() []botc.Feature                     { return nil }
func (m *mockMsgContext) SupportsFeature(f botc.Feature) bool         { return false }
func (m *mockMsgContext) Protocol() string                             { return m.protocol }
func (m *mockMsgContext) BotContext() botc.BotContext                  { return m.botCtx }
func (m *mockMsgContext) String() string                               { return m.baseMsg.Content }
func (m *mockMsgContext) Message() *botc.BaseMessage                   { return m.baseMsg }
func (m *mockMsgContext) SenderID() string                             { return m.senderID }
func (m *mockMsgContext) NewMessageBuilder() botc.MessageBuilder       { return nil }
func (m *mockMsgContext) Reply(elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	return nil, nil
}
func (m *mockMsgContext) ReplyText(a ...any) (*botc.BaseMessage, error) {
	return nil, nil
}

func TestStringListUnmarshal(t *testing.T) {
	type TestConfig struct {
		Admin map[string]StringList `json:"admin"`
	}

	// 1. 测试字符串数组
	jsonArray := `{"admin": {"qq": ["10001", "10002"]}}`
	var cfgArray TestConfig
	if err := json.Unmarshal([]byte(jsonArray), &cfgArray); err != nil {
		t.Fatalf("Unmarshal jsonArray failed: %v", err)
	}
	if len(cfgArray.Admin["qq"]) != 2 || cfgArray.Admin["qq"][0] != "10001" || cfgArray.Admin["qq"][1] != "10002" {
		t.Fatalf("unexpected admin list: %v", cfgArray.Admin["qq"])
	}

	// 2. 测试单字符串
	jsonSingle := `{"admin": {"qq": "10001"}}`
	var cfgSingle TestConfig
	if err := json.Unmarshal([]byte(jsonSingle), &cfgSingle); err != nil {
		t.Fatalf("Unmarshal jsonSingle failed: %v", err)
	}
	if len(cfgSingle.Admin["qq"]) != 1 || cfgSingle.Admin["qq"][0] != "10001" {
		t.Fatalf("unexpected single admin: %v", cfgSingle.Admin["qq"])
	}
}

func TestResolveSenderAuthority(t *testing.T) {
	grb := Create()
	grb.config.Owner = map[string]string{
		"qq":                     "100000000",
		"telegram:bot1":          "telegram:user&99999",
		"qbot:102000000":         "openid_owner",
	}
	grb.config.Admin = map[string]StringList{
		"qq":       {"100000001", "100000002"},
		"telegram": {"88888"},
	}

	tests := []struct {
		name              string
		botCtxID          string
		protocol          string
		senderID          string
		initialAuthority  entity.Authority
		expectedAuthority entity.Authority
	}{
		{
			name:              "默认值修复（0值修复为Member）",
			botCtxID:          "telegram:bot1",
			protocol:          "telegram",
			senderID:          "telegram:user&11111",
			initialAuthority:  entity.Banned, // 0值
			expectedAuthority: entity.Member, // 修复为 1
		},
		{
			name:              "普通成员保持 Member",
			botCtxID:          "onebot",
			protocol:          "onebot",
			senderID:          "onebot:user&200000000",
			initialAuthority:  entity.Member,
			expectedAuthority: entity.Member,
		},
		{
			name:              "群管理员非 Admin/Owner 保持 GroupAdmin",
			botCtxID:          "onebot",
			protocol:          "onebot",
			senderID:          "onebot:user&200000000",
			initialAuthority:  entity.GroupAdmin,
			expectedAuthority: entity.GroupAdmin,
		},
		{
			name:              "群主非 Admin/Owner 保持 GroupOwner",
			botCtxID:          "onebot",
			protocol:          "onebot",
			senderID:          "onebot:user&200000000",
			initialAuthority:  entity.GroupOwner,
			expectedAuthority: entity.GroupOwner,
		},
		{
			name:              "QQ (OneBot) 匹配别名自动赋予 Admin (等级 4)",
			botCtxID:          "onebot",
			protocol:          "onebot",
			senderID:          "onebot:user&100000001",
			initialAuthority:  entity.Member,
			expectedAuthority: entity.Admin, // 4
		},
		{
			name:              "群管如果是配置的 Admin，提升到 Admin (等级 4)",
			botCtxID:          "onebot",
			protocol:          "onebot",
			senderID:          "onebot:user&100000002",
			initialAuthority:  entity.GroupAdmin,
			expectedAuthority: entity.Admin, // 4
		},
		{
			name:              "Telegram 匹配 Admin (等级 4)",
			botCtxID:          "telegram:bot1",
			protocol:          "telegram",
			senderID:          "telegram:user&88888",
			initialAuthority:  entity.Member,
			expectedAuthority: entity.Admin, // 4
		},
		{
			name:              "QQ (Lagrange) 匹配 Owner (等级 5)",
			botCtxID:          "qq",
			protocol:          "lagrange",
			senderID:          "lagrange:user&100000000",
			initialAuthority:  entity.Member,
			expectedAuthority: entity.Owner, // 5
		},
		{
			name:              "Telegram 匹配精确 Universal ID 赋予 Owner (等级 5)",
			botCtxID:          "telegram:bot1",
			protocol:          "telegram",
			senderID:          "telegram:user&99999",
			initialAuthority:  entity.Member,
			expectedAuthority: entity.Owner, // 5
		},
		{
			name:              "QBot 匹配 Owner (等级 5)",
			botCtxID:          "qbot:102000000",
			protocol:          "qbot",
			senderID:          "qbot:user&openid_owner",
			initialAuthority:  entity.Member,
			expectedAuthority: entity.Owner, // 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgCtx := &mockMsgContext{
				botCtx:   &mockBotContext{id: tt.botCtxID},
				protocol: tt.protocol,
				senderID: tt.senderID,
				baseMsg: &botc.BaseMessage{
					Content: "test message",
					Sender: &entity.Sender{
						User: &entity.User{
							Base: &entity.Base{
								ID: tt.senderID,
							},
							Authority: tt.initialAuthority,
						},
					},
				},
			}

			grb.ResolveSenderAuthority(msgCtx)

			actual := msgCtx.Message().Sender.Authority
			if actual != tt.expectedAuthority {
				t.Errorf("expected authority %d, got %d", tt.expectedAuthority, actual)
			}

			// 测试 command.Context.Authority()
			cmdCtx := command.NewCommandContext(msgCtx, "/test")
			if cmdCtx.Authority() != tt.expectedAuthority {
				t.Errorf("cmdCtx.Authority() expected %d, got %d", tt.expectedAuthority, cmdCtx.Authority())
			}
		})
	}
}
