package telegram

import (
	"fmt"

	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/go-telegram/bot/models"
)

type MessageContext struct {
	service *Service
	msg     *models.Message
	base    *botc.BaseMessage
}

func NewMessageContext(msg *models.Message, service *Service) *MessageContext {
	return &MessageContext{
		service: service,
		msg:     msg,
	}
}

func (m *MessageContext) Protocol() string {
	return "telegram"
}

func (m *MessageContext) BotContext() botc.BotContext {
	return m.service
}

func (m *MessageContext) String() string {
	return m.Message().Content
}

func (m *MessageContext) Message() *botc.BaseMessage {
	if m.base == nil {
		m.base = ParseMessage(m.msg, m.service)
	}
	return m.base
}

func (m *MessageContext) SenderID() string {
	if m.msg.From == nil {
		return ""
	}
	return genUserID(m.msg.From.ID)
}

func (m *MessageContext) NewMessageBuilder() botc.MessageBuilder {
	return &MessageBuilder{service: m.service}
}

func (m *MessageContext) Reply(elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	if elements == nil {
		return nil, fmt.Errorf("elements is nil")
	}

	if m.Message().MessageType == botc.GroupMessage {
		return m.service.SendGroupMessage(entity.Group{
			Base: &entity.Base{
				ID:   genGroupID(m.msg.Chat.ID),
				Name: m.msg.Chat.Title,
			},
		}, elements)
	}

	return m.service.SendDirectMessage(entity.User{
		Base: &entity.Base{
			ID:   genUserID(m.msg.Chat.ID),
			Name: m.msg.Chat.Username,
		},
	}, elements)
}

func (m *MessageContext) ReplyText(a ...any) (*botc.BaseMessage, error) {
	return m.NewMessageBuilder().Text(fmt.Sprint(a...)).ReplyTo(m)
}
