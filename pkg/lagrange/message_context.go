package lagrange

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
	LgrMessage "github.com/LagrangeDev/LagrangeGo/message"
	"strconv"
	"time"
)

type MessageContext struct {
	messageType message.Type
	service     *Service
	privateMsg  *LgrMessage.PrivateMessage
	groupMsg    *LgrMessage.GroupMessage
}

func (m *MessageContext) Protocol() string {
	return "lagrange"
}

func NewMessageContext(msg any, service *Service) *MessageContext {
	switch msg := msg.(type) {
	case *LgrMessage.PrivateMessage:
		return &MessageContext{
			messageType: message.DirectMessage,
			service:     service,
			privateMsg:  msg,
		}
	case *LgrMessage.GroupMessage:
		return &MessageContext{
			messageType: message.GroupMessage,
			service:     service,
			groupMsg:    msg,
		}
	}
	return nil
}

func (m *MessageContext) String() string {
	switch m.messageType {
	case message.DirectMessage:
		return LgrMessage.ToReadableString(m.privateMsg.Elements)
	case message.GroupMessage:
		return LgrMessage.ToReadableString(m.groupMsg.Elements)
	}
	return ""
}

func (m *MessageContext) Message() *message.Base {
	switch m.messageType {
	case message.DirectMessage:
		return &message.Base{
			MessageType: m.messageType,
			ID:          strconv.FormatUint(uint64(m.privateMsg.ID), 10),
			Content:     m.String(),
			Elements:    m.service.FromMessageElements(m.privateMsg.Elements, m.privateMsg),
			Sender:      SenderConv(m.Sender(), m.groupMsg),
			Time:        time.Unix(int64(m.privateMsg.Time), 0),
		}
	case message.GroupMessage:
		return &message.Base{
			MessageType: m.messageType,
			ID:          strconv.FormatUint(uint64(m.groupMsg.ID), 10),
			Content:     m.String(),
			Elements:    m.service.FromMessageElements(m.groupMsg.Elements, m.groupMsg),
			Sender:      SenderConv(m.Sender(), m.groupMsg),
			Time:        time.Unix(int64(m.groupMsg.Time), 0),
		}
	}
	return nil
}

func (m *MessageContext) OriginalElements() []LgrMessage.IMessageElement {
	switch m.messageType {
	case message.DirectMessage:
		return m.privateMsg.Elements
	case message.GroupMessage:
		return m.groupMsg.Elements
	}
	return nil
}

func (m *MessageContext) Sender() *LgrMessage.Sender {
	switch m.messageType {
	case message.DirectMessage:
		return m.privateMsg.Sender
	case message.GroupMessage:
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

func (m *MessageContext) Reply(msg []*message.Element) error {
	switch m.messageType {
	case message.DirectMessage:
		if _, err := m.service.qqClient.SendPrivateMessage(m.privateMsg.Sender.Uin, FromBaseMessage(msg)); err != nil {
			return err
		}
	case message.GroupMessage:
		if _, err := m.service.qqClient.SendGroupMessage(m.groupMsg.GroupUin, FromBaseMessage(msg)); err != nil {
			return err
		}
	}
	return nil
}

func (m *MessageContext) ReplyText(text string) error {
	return m.Reply([]*message.Element{{
		Type:    message.Text,
		Content: text,
	}})
}
