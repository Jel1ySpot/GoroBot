package qbot

import (
	"encoding/json"
	"strings"

	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
)

// handleWebhookDispatch 处理并分发网关与 Webhook 事件
func (s *Service) handleWebhookDispatch(payload *WSPayload) error {
	switch payload.T {
	case "C2C_MESSAGE_CREATE":
		var ev C2CMessageEvent
		if err := json.Unmarshal(payload.D, &ev); err != nil {
			return err
		}
		msg := &Message{
			ID:        ev.ID,
			Content:   ev.Content,
			Timestamp: ev.Timestamp,
			Author: &User{
				ID:         ev.Author.UserOpenID,
				UserOpenID: ev.Author.UserOpenID,
			},
			DirectMessage: true,
			Attachments:   ev.Attachments,
			MsgType:       ev.MessageType,
		}
		return s.emitMessage(payload, msg)

	case "GROUP_AT_MESSAGE_CREATE", "GROUP_MESSAGE_CREATE":
		var ev GroupMessageEvent
		if err := json.Unmarshal(payload.D, &ev); err != nil {
			return err
		}
		msg := &Message{
			ID:        ev.ID,
			Content:   ev.Content,
			Timestamp: ev.Timestamp,
			GroupID:   ev.GroupOpenID,
			Author: &User{
				ID:           ev.Author.MemberOpenID,
				Username:     ev.Author.Username,
				Bot:          ev.Author.Bot,
				MemberOpenID: ev.Author.MemberOpenID,
			},
			Attachments: ev.Attachments,
			Mentions:    ev.Mentions,
			MsgType:     ev.MessageType,
		}
		return s.emitMessage(payload, msg)

	case "AT_MESSAGE_CREATE":
		var ev ChannelMessageEvent
		if err := json.Unmarshal(payload.D, &ev); err != nil {
			return err
		}
		msg := &Message{
			ID:               ev.ID,
			Content:          ev.Content,
			Timestamp:        ev.Timestamp,
			ChannelID:        ev.ChannelID,
			GuildID:          ev.GuildID,
			Author:           ev.Author,
			Attachments:      ev.Attachments,
			Mentions:         ev.Mentions,
			MentionEveryone:  ev.MentionEveryone,
			MessageReference: ev.MessageReference,
		}
		return s.emitMessage(payload, msg)

	case "DIRECT_MESSAGE_CREATE":
		var ev ChannelMessageEvent
		if err := json.Unmarshal(payload.D, &ev); err != nil {
			return err
		}
		msg := &Message{
			ID:               ev.ID,
			Content:          ev.Content,
			Timestamp:        ev.Timestamp,
			GuildID:          ev.GuildID,
			Author:           ev.Author,
			DirectMessage:    true,
			Attachments:      ev.Attachments,
			MessageReference: ev.MessageReference,
		}
		return s.emitMessage(payload, msg)
	}
	return nil
}

func (s *Service) emitMessage(event *WSPayload, data *Message) error {
	data.Content = strings.TrimSpace(data.Content)
	if strings.HasPrefix(data.Content, "/") {
		return s.emitCommand(event, data)
	}
	if s.grb != nil {
		return s.grb.MessageEmit(NewMessageContext(s, event, data))
	}
	return nil
}

func (s *Service) emitCommand(event *WSPayload, data *Message) error {
	if s.grb != nil {
		s.grb.CommandEmit(command.NewCommandContext(NewMessageContext(s, event, data), data.Content[1:]))
	}
	return nil
}
