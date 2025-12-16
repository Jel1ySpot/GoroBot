package qbot

import (
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/tencent-connect/botgo/dto"
	urlpkg "net/url"
	"path"
	"strings"
	"unsafe"
)

const (
	DirectMessage botc.MessageType = iota
	GroupMessage
	GuildMessage
)

type Message struct {
	event *dto.WSPayload
	data  *dto.Message
}

func (m *Message) ToBase(grb *GoroBot.Instant) *botc.BaseMessage {
	e := m.data
	b := botc.NewBuilder()
	if e.MessageReference != nil {
		b.Quote(&botc.BaseMessage{ID: FormatID("msg", e.MessageReference.GetEventID(), e.MessageReference.MessageID)})
	}
	if e.MentionEveryone {
		b.Mention(FormatID("user", "everyone"))
	}
	for _, user := range e.Mentions {
		b.Mention(FormatID("user", user.ID))
	}
	if e.Content != "" {
		b.Text(m.data.Content)
	}

	for _, attachment := range e.Attachments {
		if strings.HasPrefix(attachment.ContentType, "image") {
			refLink := urlpkg.Values{
				"url": {attachment.URL},
				"ext": {strings.TrimPrefix(path.Ext(attachment.URL), ".")},
			}.Encode()
			id := grb.SaveResourceLink("qbot", refLink)
			b.Append(botc.ImageElement, "[图片]", id)
		} else if strings.HasPrefix(attachment.ContentType, "video") {
			refLink := urlpkg.Values{
				"url": {attachment.URL},
				"ext": {strings.TrimPrefix(path.Ext(attachment.URL), ".")},
			}.Encode()
			id := grb.SaveResourceLink("qbot", refLink)
			b.Append(botc.VideoElement, "[视频]", id)
		} else if strings.HasPrefix(attachment.ContentType, "voice") {
			refLink := urlpkg.Values{
				"url": {attachment.URL},
				"ext": {strings.TrimPrefix(path.Ext(attachment.URL), ".")},
			}.Encode()
			id := grb.SaveResourceLink("qbot", refLink)
			b.Append(botc.VoiceElement, "[语音]", id)
		}
	}

	t, _ := e.Timestamp.Time()
	return &botc.BaseMessage{
		MessageType: botc.MessageType(1 - int(*(*byte)(unsafe.Pointer(&e.DirectMessage)))),
		ID:          e.ID,
		Content:     e.Content,
		Elements:    b.Build(),
		Sender:      Sender(m),
		Time:        t,
	}
}

func Sender(message *Message) *entity.Sender {
	if message.data.Author == nil {
		return nil
	}
	sender := entity.Sender{
		User: &entity.User{
			Base: &entity.Base{
				ID:     FormatID("user", message.data.Author.ID),
				Name:   message.data.Author.Username,
				Avatar: message.data.Author.Avatar,
			},
		},
	}
	if message.data.GroupID != "" {
		sender.From = &entity.Base{
			ID:   FormatID("group", message.data.GroupID),
			Name: message.data.GroupID,
		}
	}
	if message.data.GuildID != "" {
		sender.From = &entity.Base{
			ID:   FormatID("guild", message.data.GuildID),
			Name: message.data.GuildID,
		}
	}
	if message.data.ChannelID != "" {
		sender.From = &entity.Base{
			ID:   FormatID("channel", message.data.ChannelID),
			Name: message.data.ChannelID,
		}
	}
	return &sender
}
