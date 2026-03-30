package telegram

import (
	"net/url"
	"path"
	"time"

	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/go-telegram/bot/models"
)

// ParseMessage 将 Telegram Message 转换为 BaseMessage
func ParseMessage(msg *models.Message, s *Service) *botc.BaseMessage {
	if msg == nil {
		return nil
	}

	builder := botc.NewBuilder()

	if msg.Text != "" {
		builder.Text(msg.Text)
	}

	if msg.Caption != "" {
		builder.Text(msg.Caption)
	}

	if len(msg.Photo) > 0 {
		photo := largestPhoto(msg.Photo)
		refLink := url.Values{
			"file_id": {photo.FileID},
			"ext":     {"jpg"},
		}.Encode()
		id := s.grb.SaveResourceLink(s.ID(), refLink)
		builder.Append(botc.ImageElement, "[图片]", id)
	}

	if msg.Document != nil {
		ext := path.Ext(msg.Document.FileName)
		refLink := url.Values{
			"file_id": {msg.Document.FileID},
		}
		if ext != "" {
			refLink.Set("ext", ext[1:])
		}
		id := s.grb.SaveResourceLink(s.ID(), refLink.Encode())
		builder.Append(botc.FileElement, msg.Document.FileName, id)
	}

	messageType := botc.DirectMessage
	if msg.Chat.Type == models.ChatTypeGroup || msg.Chat.Type == models.ChatTypeSupergroup {
		messageType = botc.GroupMessage
	}

	return &botc.BaseMessage{
		MessageType: messageType,
		ID:          genMessageID(msg.Chat.ID, msg.ID),
		Content:     botc.ElemsToString(builder.Build()),
		Elements:    builder.Build(),
		Sender:      senderFromMessage(msg),
		Time:        time.Unix(int64(msg.Date), 0),
	}
}

func senderFromMessage(msg *models.Message) *entity.Sender {
	if msg.From == nil {
		return nil
	}

	sender := entity.Sender{
		User: &entity.User{
			Base: &entity.Base{
				ID:   genUserID(msg.From.ID),
				Name: chooseName(msg.From.Username, msg.From.FirstName),
			},
			Nickname: msg.From.FirstName,
		},
	}

	if msg.Chat.Type == models.ChatTypeGroup || msg.Chat.Type == models.ChatTypeSupergroup {
		sender.From = &entity.Base{
			ID:   genGroupID(msg.Chat.ID),
			Name: chooseName(msg.Chat.Title, msg.Chat.Username),
		}
	}
	return &sender
}

func largestPhoto(list []models.PhotoSize) models.PhotoSize {
	var chosen models.PhotoSize
	var size int
	for _, p := range list {
		if p.FileSize > size {
			chosen = p
			size = p.FileSize
		}
	}
	if chosen.FileID == "" && len(list) > 0 {
		return list[len(list)-1]
	}
	return chosen
}

func chooseName(names ...string) string {
	for _, n := range names {
		if n != "" {
			return n
		}
	}
	return ""
}
