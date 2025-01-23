package lagrange

import (
	"fmt"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	LgrMessage "github.com/LagrangeDev/LagrangeGo/message"
)

type MessageContext struct {
	messageType botc.MessageType
	service     *Service
	base        *botc.BaseMessage
	privateMsg  *LgrMessage.PrivateMessage
	groupMsg    *LgrMessage.GroupMessage
}

func (m *MessageContext) BotContext() botc.BotContext {
	return &Context{m.service}
}

func (m *MessageContext) Protocol() string {
	return "lagrange"
}

func NewMessageContext(msg any, service *Service) *MessageContext {
	switch msg := msg.(type) {
	case *LgrMessage.PrivateMessage:
		return &MessageContext{
			messageType: botc.DirectMessage,
			service:     service,
			privateMsg:  msg,
		}
	case *LgrMessage.GroupMessage:
		return &MessageContext{
			messageType: botc.GroupMessage,
			service:     service,
			groupMsg:    msg,
		}
	}
	return nil
}

func (m *MessageContext) String() string {
	switch m.messageType {
	case botc.DirectMessage:
		return LgrMessage.ToReadableString(m.privateMsg.Elements)
	case botc.GroupMessage:
		return LgrMessage.ToReadableString(m.groupMsg.Elements)
	}
	return ""
}

func (m *MessageContext) Message() *botc.BaseMessage {
	if m.base == nil {
		switch m.messageType {
		case botc.DirectMessage:
			m.base, _ = ParseMessageEvent(m.service, m.privateMsg)
		case botc.GroupMessage:
			m.base, _ = ParseMessageEvent(m.service, m.groupMsg)
		}
	}

	return m.base
}

func (m *MessageContext) SenderID() string {
	switch m.messageType {
	case botc.DirectMessage:
		return fmt.Sprintf("%d", m.privateMsg.Sender.Uin)
	case botc.GroupMessage:
		return fmt.Sprintf("%d", m.groupMsg.Sender.Uin)
	}
	return ""
}

func (m *MessageContext) NewMessageBuilder() botc.MessageBuilder {
	return &MessageBuilder{
		service: m.service,
	}
}

func (m *MessageContext) OriginalElements() []LgrMessage.IMessageElement {
	switch m.messageType {
	case botc.DirectMessage:
		return m.privateMsg.Elements
	case botc.GroupMessage:
		return m.groupMsg.Elements
	}
	return nil
}

func (m *MessageContext) Sender() *LgrMessage.Sender {
	switch m.messageType {
	case botc.DirectMessage:
		return m.privateMsg.Sender
	case botc.GroupMessage:
		return m.groupMsg.Sender
	}
	return nil
}

func (m *MessageContext) GroupUin() uint32 {
	if m.groupMsg != nil {
		return m.groupMsg.GroupUin
	}
	return 0
}

func (m *MessageContext) reply(elements []LgrMessage.IMessageElement) (*botc.BaseMessage, error) {
	if elements == nil {
		return nil, fmt.Errorf("elements is nil")
	}
	switch m.messageType {
	case botc.DirectMessage:
		if msg, err := m.service.qqClient.SendPrivateMessage(m.privateMsg.Sender.Uin, elements); err != nil {
			return nil, err
		} else {
			return ParseMessageEvent(m.service, msg)
		}
	case botc.GroupMessage:
		if msg, err := m.service.qqClient.SendGroupMessage(m.groupMsg.GroupUin, elements); err != nil {
			return nil, err
		} else {
			return ParseMessageEvent(m.service, msg)
		}
	}
	return nil, fmt.Errorf("unhandled message type: %v", m.messageType)
}

func (m *MessageContext) Reply(msg []*botc.MessageElement) (*botc.BaseMessage, error) {
	return m.reply(TranslateMessageElement(m.service, msg))
}

func (m *MessageContext) ReplyText(a ...any) (*botc.BaseMessage, error) {
	return m.reply([]LgrMessage.IMessageElement{
		LgrMessage.NewText(fmt.Sprint(a...)),
	})
}
