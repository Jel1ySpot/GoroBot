package qbot

import (
	"context"
	"fmt"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/tencent-connect/botgo/dto"
)

type MessageContext struct {
	bot     *Context
	message *Message
}

func (m *MessageContext) Protocol() string {
	return "qbot"
}

func (m *MessageContext) BotContext() botc.BotContext {
	return m.bot
}

func (m *MessageContext) String() string {
	return m.message.data.Content
}

func (m *MessageContext) Message() *botc.BaseMessage {
	return m.message.ToBase(m.bot.Service.grb)
}

func (m *MessageContext) SenderID() string {
	return m.message.data.Author.ID
}

func (m *MessageContext) NewMessageBuilder() botc.MessageBuilder {
	return NewMessageBuilder(m, m.bot.Service)
}

func (m *MessageContext) Reply(elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	return nil, nil
}

func (m *MessageContext) reply(body *dto.MessageToCreate) (*botc.BaseMessage, error) {
	if m.message.event != nil {
		body.EventID = m.message.event.EventID
	}
	body.MsgID = m.message.data.ID
	if m.message.data.DirectMessage {
		msg, err := m.bot.api.PostC2CMessage(context.Background(), m.message.data.Author.ID, body)
		if err != nil {
			return nil, err
		}
		msgCtx := Message{data: msg}
		return msgCtx.ToBase(m.bot.grb), nil
	}
	if m.message.data.GroupID != "" {
		msg, err := m.bot.api.PostGroupMessage(context.Background(), m.message.data.GroupID, body)
		if err != nil {
			return nil, err
		}
		msgCtx := Message{data: msg}
		return msgCtx.ToBase(m.bot.grb), nil
	}
	if m.message.data.ChannelID != "" {
		msg, err := m.bot.api.PostMessage(context.Background(), m.message.data.ChannelID, body)
		if err != nil {
			return nil, err
		}
		msgCtx := Message{data: msg}
		return msgCtx.ToBase(m.bot.grb), nil
	}
	return nil, nil
}

func (m *MessageContext) ReplyText(a ...any) (*botc.BaseMessage, error) {
	return m.NewMessageBuilder().Text(fmt.Sprint(a...)).ReplyTo(m)
}
