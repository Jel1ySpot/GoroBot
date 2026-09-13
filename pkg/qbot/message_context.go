package qbot

import (
	"context"
	"fmt"
	urlpkg "net/url"
	"path"
	"strings"
	"time"

	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
)

const (
	DirectMessage botc.MessageType = iota
	GroupMessage
	GuildMessage
)

type MessageContext struct {
	bot   *Service
	base  *botc.BaseMessage
	event *WSPayload
	data  *Message
}

func NewMessageContext(bot *Service, event *WSPayload, data *Message) *MessageContext {
	return &MessageContext{
		bot:   bot,
		event: event,
		data:  data,
	}
}

func (m *MessageContext) Protocol() string {
	return "qbot"
}

func (m *MessageContext) BotContext() botc.BotContext {
	return m.bot
}

func (m *MessageContext) Features() []botc.Feature {
	if m.bot != nil {
		return m.bot.Features()
	}
	return nil
}

func (m *MessageContext) SupportsFeature(feature botc.Feature) bool {
	if m.bot != nil {
		return m.bot.SupportsFeature(feature)
	}
	return false
}

func (m *MessageContext) String() string {
	return m.data.Content
}

func (m *MessageContext) Message() *botc.BaseMessage {
	if m.base == nil {
		m.base = ParseMessage(m.bot.grb, m.bot, m.data)
	}
	return m.base
}

func (m *MessageContext) SenderID() string {
	if m.data.Author != nil {
		return m.data.Author.ID
	}
	return ""
}

func (m *MessageContext) NewMessageBuilder() botc.MessageBuilder {
	return NewMessageBuilder(m)
}

func (m *MessageContext) Reply(elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	builder := NewMessageBuilder(m)
	builder.ApplyElements(elements)
	return builder.ReplyTo(m)
}

func (m *MessageContext) reply(body *MessageToCreate) (*botc.BaseMessage, error) {
	if m.event != nil {
		body.EventID = m.event.ID
	}
	body.MsgID = m.data.ID

	if m.data.DirectMessage && m.data.Author != nil {
		msg, err := m.bot.api.PostC2CMessage(context.Background(), m.data.Author.ID, body)
		if err != nil {
			return nil, err
		}
		return ParseMessage(m.bot.grb, m.bot, msg), nil
	}
	if m.data.GroupID != "" {
		msg, err := m.bot.api.PostGroupMessage(context.Background(), m.data.GroupID, body)
		if err != nil {
			return nil, err
		}
		return ParseMessage(m.bot.grb, m.bot, msg), nil
	}
	if m.data.ChannelID != "" {
		msg, err := m.bot.api.PostMessage(context.Background(), m.data.ChannelID, body)
		if err != nil {
			return nil, err
		}
		return ParseMessage(m.bot.grb, m.bot, msg), nil
	}
	return nil, nil
}

func (m *MessageContext) ReplyText(a ...any) (*botc.BaseMessage, error) {
	return m.NewMessageBuilder().Text(fmt.Sprint(a...)).ReplyTo(m)
}

// ParseMessage 将 Message 转换为 BaseMessage
func ParseMessage(grb *GoroBot.Instant, bot *Service, data *Message) *botc.BaseMessage {
	b := botc.NewBuilder()
	if data.MessageReference != nil {
		b.Quote(&botc.BaseMessage{ID: FormatID("msg", data.MessageReference.EventID, data.MessageReference.MessageID)})
	}
	if data.MentionEveryone {
		b.Mention(FormatID("user", "everyone"))
	}
	for _, user := range data.Mentions {
		b.Mention(FormatID("user", user.ID))
	}
	if data.Content != "" {
		b.Text(data.Content)
	}

	for _, attachment := range data.Attachments {
		if strings.HasPrefix(attachment.ContentType, "image") {
			refLink := urlpkg.Values{
				"url": {attachment.URL},
				"ext": {strings.TrimPrefix(path.Ext(attachment.URL), ".")},
			}.Encode()
			id := grb.SaveResourceLink(bot.ID(), refLink)
			b.Append(botc.ImageElement, "[图片]", id)
		} else if strings.HasPrefix(attachment.ContentType, "video") {
			refLink := urlpkg.Values{
				"url": {attachment.URL},
				"ext": {strings.TrimPrefix(path.Ext(attachment.URL), ".")},
			}.Encode()
			id := grb.SaveResourceLink(bot.ID(), refLink)
			b.Append(botc.VideoElement, "[视频]", id)
		} else if strings.HasPrefix(attachment.ContentType, "voice") {
			refLink := urlpkg.Values{
				"url": {attachment.URL},
				"ext": {strings.TrimPrefix(path.Ext(attachment.URL), ".")},
			}.Encode()
			id := grb.SaveResourceLink(bot.ID(), refLink)
			b.Append(botc.VoiceElement, "[语音]", id)
		}
	}

	msgType := botc.MessageType(0)
	if data.DirectMessage {
		msgType = DirectMessage
	} else if data.GroupID != "" {
		msgType = GroupMessage
	} else {
		msgType = GuildMessage
	}

	var msgTime time.Time
	if data.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, data.Timestamp); err == nil {
			msgTime = t
		}
	}
	if msgTime.IsZero() {
		msgTime = time.Now()
	}

	return &botc.BaseMessage{
		MessageType: msgType,
		ID:          data.ID,
		Content:     data.Content,
		Elements:    b.Build(),
		Sender:      parseSender(data),
		Time:        msgTime,
	}
}

func parseSender(data *Message) *entity.Sender {
	if data.Author == nil {
		return nil
	}
	sender := entity.Sender{
		User: &entity.User{
			Base: &entity.Base{
				ID:     FormatID("user", data.Author.ID),
				Name:   data.Author.Username,
				Avatar: data.Author.Avatar,
			},
		},
	}
	if data.GroupID != "" {
		sender.From = &entity.Base{
			ID:   FormatID("group", data.GroupID),
			Name: data.GroupID,
		}
	}
	if data.GuildID != "" {
		sender.From = &entity.Base{
			ID:   FormatID("guild", data.GuildID),
			Name: data.GuildID,
		}
	}
	if data.ChannelID != "" {
		sender.From = &entity.Base{
			ID:   FormatID("channel", data.ChannelID),
			Name: data.ChannelID,
		}
	}
	return &sender
}
