package lagrange

import (
	"fmt"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	LgrMessage "github.com/LagrangeDev/LagrangeGo/message"
	"strconv"
	"time"
)

func ParseElementsFromEvent(service *Service, msgEvent any) []*botc.MessageElement {
	b := botc.NewBuilder()
	var elements []LgrMessage.IMessageElement
	switch e := msgEvent.(type) {
	case *LgrMessage.PrivateMessage:
		elements = e.Elements
	case *LgrMessage.GroupMessage:
		elements = e.Elements
	default:
		return nil
	}
	for _, elem := range elements {
		switch elem := elem.(type) {
		case *LgrMessage.TextElement:
			b.Append(botc.TextElement, elem.Content, "")
		case *LgrMessage.AtElement:
			b.Append(
				botc.MentionElement,
				elem.Display,
				GenUserID(elem.TargetUin),
			)
		case *LgrMessage.FaceElement:
			b.Append(
				botc.StickerElement,
				"[表情]",
				strconv.FormatUint(uint64(elem.FaceID), 10),
			)
		case *LgrMessage.ReplyElement:
			b.Append(
				botc.QuoteElement,
				"[回复]",
				ReplyElementToMessage(service, elem).Marshall(),
			)
		case *LgrMessage.VoiceElement:
			var (
				url string
				err error
			)
			switch msgEvent := msgEvent.(type) {
			case *LgrMessage.GroupMessage:
				url, err = service.qqClient.GetGroupRecordURL(msgEvent.GroupUin, elem.Node)
				if err == nil {
					_, err = service.grb.SaveRemoteResource(url)
				}
			case *LgrMessage.PrivateMessage:
				url, err = service.qqClient.GetPrivateRecordURL(elem.Node)
				if err == nil {
					_, err = service.grb.SaveRemoteResource(url)
				}
			}
			if err != nil {
				service.logger.Warning("save voice err: %v", err)
			}

			b.Append(
				botc.VoiceElement,
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
				if CheckMessageType(msgEvent) == botc.GroupMessage {
					imageElem, err = service.qqClient.QueryGroupImage(elem.Md5, elem.FileUUID)
				} else {
					imageElem, err = service.qqClient.QueryFriendImage(elem.Md5, elem.FileUUID)
				}
				if err == nil {
					url = imageElem.URL
				}
			}

			if err == nil {
				_, err = service.grb.SaveRemoteResource(url)
			}
			if err != nil {
				service.logger.Warning("save image err: %v", err)
			}

			b.Append(
				botc.ImageElement,
				func() string {
					if elem.Summary != "" {
						return elem.Summary
					}
					if elem.SubType == 7 {
						return "[动画表情]"
					}
					return "[图片]"
				}(),
				fmt.Sprintf("%x", elem.Md5),
			)
		case *LgrMessage.FileElement:
			b.Append(botc.FileElement,
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
			url, err := service.qqClient.GetVideoURL(CheckMessageType(msgEvent) == botc.GroupMessage, elem.UUID)
			if err == nil {
				_, err = service.grb.SaveRemoteResource(url)
			}
			if err != nil {
				service.logger.Warning("save short video err: %v", err)
			}
			b.Append(botc.VideoElement,
				"[视频]",
				fmt.Sprintf("%x", elem.Md5),
			)
		}
	}
	return b.Build()
}

func ParseMessageEvent(service *Service, msgEvent any) (*botc.BaseMessage, error) {
	elems := ParseElementsFromEvent(service, msgEvent)
	content := botc.ElemsToString(elems)
	switch msgEvent := msgEvent.(type) {
	case *LgrMessage.PrivateMessage:
		return &botc.BaseMessage{
			MessageType: botc.DirectMessage,
			ID:          GenMsgSeqID(msgEvent.ID),
			Content:     content,
			Elements:    elems,
			Sender:      SenderConv(msgEvent.Sender, nil),
			Time:        time.Unix(int64(msgEvent.Time), 0),
		}, nil
	case *LgrMessage.GroupMessage:
		return &botc.BaseMessage{
			MessageType: botc.GroupMessage,
			ID:          GenMsgSeqID(msgEvent.ID),
			Content:     content,
			Elements:    elems,
			Sender:      SenderConv(msgEvent.Sender, msgEvent),
			Time:        time.Unix(int64(msgEvent.Time), 0),
		}, nil
	}
	return nil, fmt.Errorf("unhandled message type: %T", msgEvent)
}

func TranslateMessageElement(service *Service, elements []*botc.MessageElement) []LgrMessage.IMessageElement {
	b := MessageBuilder{}
	for _, elem := range elements {
		switch elem.Type {
		case botc.TextElement:
			b.Text(elem.Content)
		case botc.QuoteElement:
			targetMessage, err := botc.UnmarshallMessage(elem.Source)
			if err != nil {
				continue
			}
			b.Quote(targetMessage)
		case botc.MentionElement:
			b.Mention(elem.Source)
		case botc.ImageElement:
			if resource, err := service.grb.GetResource(elem.Source); err == nil {
				b.ImageFromFile(resource.FilePath)
			}
		case botc.VideoElement:
			if resource, err := service.grb.GetResource(elem.Source); err == nil {
				b.VideoFromFile(resource.FilePath)
			}
		case botc.FileElement:
			if resource, err := service.grb.GetResource(elem.Source); err == nil {
				b.File(resource.FilePath, elem.Content)
			}
		case botc.VoiceElement:
			if resource, err := service.grb.GetResource(elem.Source); err == nil {
				b.Voice(resource.FilePath)
			}
		case botc.StickerElement:
			b.Sticker(elem.Source)
		case botc.LinkElement:
			b.Text(elem.Source)
		case botc.OtherElement:
			b.Text(elem.Content)
		}
	}
	return b.Build()
}
