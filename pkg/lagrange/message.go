package lagrange

import (
	"fmt"
	urlpkg "net/url"
	"path"
	"strconv"
	"strings"
	"time"

	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	LgrMessage "github.com/LagrangeDev/LagrangeGo/message"
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
				resourceURL string
				err         error
			)
			switch msgEvent := msgEvent.(type) {
			case *LgrMessage.GroupMessage:
				resourceURL, err = service.qqClient.GetGroupRecordURL(msgEvent.GroupUin, elem.Node)
			case *LgrMessage.PrivateMessage:
				resourceURL, err = service.qqClient.GetPrivateRecordURL(elem.Node)
			}
			if err != nil {
				service.logger.Error("save voice err: %v", err)
			}

			refLink := ""
			if resourceURL != "" {
				refLink = urlpkg.Values{
					"url": {resourceURL},
					"ext": {strings.TrimPrefix(path.Ext(resourceURL), ".")},
				}.Encode()
			}
			resourceID := service.grb.SaveResourceLink(service.Protocol(), refLink)

			b.Append(
				botc.VoiceElement,
				"[录音]",
				resourceID,
			)
		case *LgrMessage.ImageElement:
			var (
				imageElem   *LgrMessage.ImageElement
				err         error
				resourceURL string
			)

			if elem.URL != "" {
				resourceURL = elem.URL
			} else {
				if CheckMessageType(msgEvent) == botc.GroupMessage {
					imageElem, err = service.qqClient.QueryGroupImage(elem.Md5, elem.FileUUID)
				} else {
					imageElem, err = service.qqClient.QueryFriendImage(elem.Md5, elem.FileUUID)
				}
				if err == nil {
					resourceURL = imageElem.URL
				}
			}

			if err != nil {
				service.logger.Error("save image err: %v", err)
			}

			refLink := ""
			if resourceURL != "" {
				refLink = urlpkg.Values{
					"url": {resourceURL},
					"ext": {strings.TrimPrefix(path.Ext(resourceURL), ".")},
				}.Encode()
			}
			resourceID := service.grb.SaveResourceLink(service.Protocol(), refLink)

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
				resourceID,
			)
		case *LgrMessage.FileElement:
			refLink := ""
			switch e := msgEvent.(type) {
			case *LgrMessage.PrivateMessage:
				if resourceURL, err := service.qqClient.GetPrivateFileURL(elem.FileUUID, elem.FileHash); err == nil {
					refLink = urlpkg.Values{
						"url": {resourceURL},
						"ext": {strings.TrimPrefix(path.Ext(resourceURL), ".")},
					}.Encode()
				} else {
					service.logger.Error("get private file url err: %v", err)
				}
			case *LgrMessage.GroupMessage:
				if resourceURL, err := service.qqClient.GetGroupFileURL(e.GroupUin, elem.FileID); err == nil {
					refLink = urlpkg.Values{
						"url": {resourceURL},
						"ext": {strings.TrimPrefix(path.Ext(resourceURL), ".")},
					}.Encode()
				} else {
					service.logger.Error("get group file url err: %v", err)
				}
			}
			resourceID := service.grb.SaveResourceLink(service.Protocol(), refLink)
			b.Append(botc.FileElement, "[文件]", resourceID)
		case *LgrMessage.ShortVideoElement:
			resourceID := ""
			switch e := msgEvent.(type) {
			case *LgrMessage.PrivateMessage:
				if resourceURL, err := service.qqClient.GetPrivateVideoURL(elem.Node); err == nil {
					refLink := urlpkg.Values{
						"url": {resourceURL},
						"ext": {strings.TrimPrefix(path.Ext(resourceURL), ".")},
					}.Encode()
					resourceID = service.grb.SaveResourceLink(service.Protocol(), refLink)
				} else {
					service.logger.Error("get private video url err: %v", err)
				}
			case *LgrMessage.GroupMessage:
				if resourceURL, err := service.qqClient.GetGroupVideoURL(e.GroupUin, elem.Node); err == nil {
					refLink := urlpkg.Values{
						"url": {resourceURL},
						"ext": {strings.TrimPrefix(path.Ext(resourceURL), ".")},
					}.Encode()
					resourceID = service.grb.SaveResourceLink(service.Protocol(), refLink)
				} else {
					service.logger.Error("get group video url err: %v", err)
				}
			default:
				continue
			}
			b.Append(botc.VideoElement,
				"[视频]",
				resourceID,
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
			if pathStr, err := service.grb.LoadResourceFromID(elem.Source); err == nil {
				b.ImageFromFile(pathStr)
			}
		case botc.VideoElement:
			if pathStr, err := service.grb.LoadResourceFromID(elem.Source); err == nil {
				b.VideoFromFile(pathStr)
			}
		case botc.FileElement:
			if pathStr, err := service.grb.LoadResourceFromID(elem.Source); err == nil {
				b.File(pathStr, elem.Content)
			}
		case botc.VoiceElement:
			if pathStr, err := service.grb.LoadResourceFromID(elem.Source); err == nil {
				b.Voice(pathStr)
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
