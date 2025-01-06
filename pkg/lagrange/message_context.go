package lagrange

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
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
			Elements:    m.service.FromMessageElements(m.privateMsg.Elements, message.DirectMessage),
			Sender: entity.Sender{
				User: SenderToUser(m.privateMsg.Sender),
			},
			Time: time.Unix(int64(m.privateMsg.Time), 0),
		}
	case message.GroupMessage:
		return &message.Base{
			MessageType: m.messageType,
			ID:          strconv.FormatUint(uint64(m.groupMsg.ID), 10),
			Content:     m.String(),
			Elements:    m.service.FromMessageElements(m.groupMsg.Elements, message.GroupMessage),
			Sender: entity.Sender{
				User: SenderToUser(m.groupMsg.Sender),
				From: strconv.FormatUint(uint64(m.groupMsg.GroupUin), 10),
			},
			Time: time.Unix(int64(m.groupMsg.Time), 0),
		}
	}
	return nil
}

func (m *MessageContext) Protocol() string {
	return "qq"
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
