package qbot

import (
	"context"
	"fmt"
	"strings"
	"unsafe"

	urlpkg "net/url"
	"path"

	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/tencent-connect/botgo/dto"
)

const (
	DirectMessage botc.MessageType = iota
	GroupMessage
	GuildMessage
)

type MessageContext struct {
	bot   *Service
	base  *botc.BaseMessage
	event *dto.WSPayload
	data  *dto.Message
}

func NewMessageContext(bot *Service, event *dto.WSPayload, data *dto.Message) *MessageContext {
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
	return m.data.Author.ID
}

func (m *MessageContext) NewMessageBuilder() botc.MessageBuilder {
	return NewMessageBuilder(m)
}

func (m *MessageContext) Reply(elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	return nil, nil
}

func (m *MessageContext) reply(body *dto.MessageToCreate) (*botc.BaseMessage, error) {
	if m.event != nil {
		body.EventID = m.event.EventID
	}
	body.MsgID = m.data.ID
	if m.data.DirectMessage {
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

// ParseMessage 将 dto.Message 转换为 BaseMessage
func ParseMessage(grb *GoroBot.Instant, bot *Service, data *dto.Message) *botc.BaseMessage {
	b := botc.NewBuilder()
	if data.MessageReference != nil {
		b.Quote(&botc.BaseMessage{ID: FormatID("msg", data.MessageReference.GetEventID(), data.MessageReference.MessageID)})
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

	t, _ := data.Timestamp.Time()
	return &botc.BaseMessage{
		MessageType: botc.MessageType(1 - int(*(*byte)(unsafe.Pointer(&data.DirectMessage)))),
		ID:          data.ID,
		Content:     data.Content,
		Elements:    b.Build(),
		Sender:      parseSender(data),
		Time:        t,
	}
}

func parseSender(data *dto.Message) *entity.Sender {
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
