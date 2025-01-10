package lagrange

import (
	"fmt"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
	LgrMessage "github.com/LagrangeDev/LagrangeGo/message"
	"strconv"
)

func (s *Service) FromMessageElements(elements []LgrMessage.IMessageElement, msgEvent any) []*message.Element {
	b := message.NewBuilder()
	for _, elem := range elements {
		switch elem := elem.(type) {
		case *LgrMessage.TextElement:
			b.Append(message.Text, elem.Content, "")
		case *LgrMessage.AtElement:
			b.Append(
				message.Mention,
				elem.Display,
				strconv.FormatUint(uint64(elem.TargetUin), 10),
			)
		case *LgrMessage.FaceElement:
			b.Append(
				message.Sticker,
				"[表情]",
				strconv.FormatUint(uint64(elem.FaceID), 10),
			)
		case *LgrMessage.ReplyElement:
			source := message.Base{
				MessageType: func() message.Type {
					if elem.GroupUin > 0 {
						return message.GroupMessage
					}
					return message.DirectMessage
				}(),
				ID:       strconv.FormatUint(uint64(elem.ReplySeq), 10),
				Content:  LgrMessage.ToReadableString(elem.Elements),
				Elements: s.FromMessageElements(elem.Elements, msgEvent),
				Sender: &entity.Sender{
					User: &entity.User{
						Base: &entity.Base{
							ID: strconv.FormatUint(uint64(elem.SenderUin), 10),
						},
					},
					From: func() string {
						if elem.GroupUin > 0 {
							return strconv.FormatUint(uint64(elem.GroupUin), 10)
						}
						return ""
					}(),
				},
			}
			b.Append(
				message.Quote,
				"[回复]",
				source.Marshall(),
			)
		case *LgrMessage.VoiceElement:
			var (
				url string
				err error
			)
			switch msgEvent := msgEvent.(type) {
			case *LgrMessage.GroupMessage:
				url, err = s.qqClient.GetGroupRecordURL(msgEvent.GroupUin, elem.Node)
				if err == nil {
					err = s.bot.SaveResource(fmt.Sprintf("%x", elem.Md5), url)
				}
			case *LgrMessage.PrivateMessage:
				url, err = s.qqClient.GetPrivateRecordURL(elem.Node)
				if err == nil {
					err = s.bot.SaveResource(fmt.Sprintf("%x", elem.Md5), url)
				}
			}
			if err != nil {
				s.logger.Warning("save voice err: %v", err)
			}

			b.Append(
				message.Voice,
				"[录音]",
				fmt.Sprintf("%x", elem.Md5),
			)
		case *LgrMessage.ImageElement:
			var (
				imageElem *LgrMessage.ImageElement
				err       error
				url       string
			)

			if elem.URL != "" {
				url = elem.URL
			} else {
				if CheckMessageType(msgEvent) == message.GroupMessage {
					imageElem, err = s.qqClient.QueryGroupImage(elem.Md5, elem.FileUUID)
				} else {
					imageElem, err = s.qqClient.QueryFriendImage(elem.Md5, elem.FileUUID)
				}
				if err == nil {
					url = imageElem.URL
				}
			}

			if err == nil {
				err = s.bot.SaveResource(fmt.Sprintf("%x", elem.Md5), url)
			}
			if err != nil {
				s.logger.Warning("save image err: %v", err)
			}

			b.Append(
				message.Image,
				"[照片]",
				fmt.Sprintf("%x", elem.Md5),
			)
		case *LgrMessage.FileElement:
			b.Append(message.File,
				"[文件]",
				func() string {
					switch e := msgEvent.(type) {
					case *LgrMessage.PrivateMessage:
						return fmt.Sprintf("lagrange:%s&%s", elem.FileUUID, elem.FileHash)
					case *LgrMessage.GroupMessage:
						return fmt.Sprintf("lagrange:%d&%s", e.GroupUin, elem.FileID)
					}
					return ""
				}(),
			)
		case *LgrMessage.ShortVideoElement:
			url, err := s.qqClient.GetVideoURL(CheckMessageType(msgEvent) == message.GroupMessage, elem.UUID)
			if err == nil {
				err = s.bot.SaveResource(fmt.Sprintf("%x", elem.Md5), url)
			}
			if err != nil {
				s.logger.Warning("save short video err: %v", err)
			}
			b.Append(message.Video,
				"[视频]",
				fmt.Sprintf("%x", elem.Md5),
			)
		}
	}
	return b.Build()
}

func (s *Service) FromBaseMessage(msg []*message.Element) []LgrMessage.IMessageElement {
	b := MessageBuilder{}
	for _, elem := range msg {
		switch elem.Type {
		case message.Text:
			b.Text(elem.Content)
		case message.Quote:
			targetMessage, err := message.UnmarshallMessage(elem.Source)
			if err != nil {
				continue
			}
			b.Quote(targetMessage)
		case message.Mention:
			b.Mention(elem.Source)
		case message.Image:
			if resource, err := s.bot.GetResource(elem.Source); err == nil {
				b.ImageFromFile(resource.FilePath)
			}
		case message.Video:
			if resource, err := s.bot.GetResource(elem.Source); err == nil {
				b.VideoFromFile(resource.FilePath)
			}
		case message.File:
			if resource, err := s.bot.GetResource(elem.Source); err == nil {
				b.File(resource.FilePath, elem.Content)
			}
		case message.Voice:
			if resource, err := s.bot.GetResource(elem.Source); err == nil {
				b.Voice(resource.FilePath)
			}
		case message.Sticker:
			b.Sticker(elem.Source)
		case message.Link:
			b.Text(elem.Source)
		case message.Other:
			b.Text(elem.Content)
		}
	}
	return b.Build()
}
