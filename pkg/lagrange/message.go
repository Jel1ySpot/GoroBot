package lagrange

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
	lgrMessage "github.com/LagrangeDev/LagrangeGo/message"
	"strconv"
)

func (s *Service) FromMessageElements(elements []lgrMessage.IMessageElement, messageType message.Type) []*message.Element {
	b := message.NewBuilder()
	for _, elem := range elements {
		switch elem.(type) {
		case *lgrMessage.TextElement:
			b.Append(message.Text, elem.(*lgrMessage.TextElement).Content, "")
		case *lgrMessage.AtElement:
			b.Append(
				message.Mention,
				elem.(*lgrMessage.AtElement).Display,
				strconv.FormatUint(uint64(elem.(*lgrMessage.AtElement).TargetUin), 10),
			)
		case *lgrMessage.FaceElement:
			b.Append(
				message.Sticker,
				"[表情]",
				strconv.FormatUint(uint64(elem.(*lgrMessage.FaceElement).FaceID), 10),
			)
		case *lgrMessage.ReplyElement:
			elem := elem.(*lgrMessage.ReplyElement)
			source := message.Base{
				MessageType: 0,
				ID:          strconv.FormatUint(uint64(elem.ReplySeq), 10),
				Content:     lgrMessage.ToReadableString(elem.Elements),
				Elements:    s.FromMessageElements(elem.Elements, messageType),
				Sender: entity.Sender{
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
		case *lgrMessage.VoiceElement:
			elem := elem.(*lgrMessage.VoiceElement)
			b.Append(
				message.Voice,
				"[录音]",
				elem.URL,
			)
		case *lgrMessage.ImageElement:
			elem := elem.(*lgrMessage.ImageElement)
			b.Append(
				message.Image,
				"[照片]",
				func() string {
					switch messageType {
					case message.DirectMessage:
						if res, err := s.qqClient.QueryFriendImage(elem.Md5, elem.FileUUID); err != nil {
							return ""
						} else {
							return res.URL
						}
					case message.GroupMessage:
						if res, err := s.qqClient.QueryGroupImage(elem.Md5, elem.FileUUID); err != nil {
							return ""
						} else {
							return res.URL
						}
					}
					return ""
				}())
		case *lgrMessage.FileElement:
			elem := elem.(*lgrMessage.FileElement)
			b.Append(message.File,
				"[文件]",
				elem.FileURL,
			)
		case *lgrMessage.ShortVideoElement:
			elem := elem.(*lgrMessage.ShortVideoElement)
			b.Append(message.Video,
				"[视频]",
				elem.URL)
		}
	}
	return b.Build()
}

func FromBaseMessage(msg []*message.Element) []lgrMessage.IMessageElement {
	b := messageBuilder{}
	for _, elem := range msg {
		switch elem.Type {
		case message.Text:
			b.text(elem.Content)
		case message.Quote:
			b.reply(elem.Source)
		case message.Mention:
			b.mention(elem.Source)
		case message.Image:
			b.image(elem.Source)
		case message.Video:
			b.video(elem.Source)
		case message.File:
			b.file(elem.Source, elem.Content)
		case message.Voice:
			b.voice(elem.Source)
		case message.Sticker:
			b.sticker(elem.Source)
		case message.Link:
			b.text(elem.Source)
		case message.Other:
			b.text(elem.Content)
		}
	}
	return b.Build()
}

type messageBuilder struct {
	elements []lgrMessage.IMessageElement
}

func (b *messageBuilder) Build() []lgrMessage.IMessageElement {
	return b.elements
}

func (b *messageBuilder) reply(source string) *messageBuilder {
	msg, err := message.UnmarshallMessage(source)
	if err != nil {
		return b
	}
	id, err := strconv.ParseUint(msg.ID, 10, 32)
	if err != nil {
		return b
	}
	sender, err := strconv.ParseUint(msg.Sender.ID, 10, 32)
	if err != nil {
		return b
	}
	switch msg.MessageType {
	case message.DirectMessage:
		b.elements = append(b.elements, lgrMessage.NewPrivateReply(&lgrMessage.PrivateMessage{
			ID:   uint32(id),
			Time: uint32(msg.Time.Unix()),
			Sender: &lgrMessage.Sender{
				Uin: uint32(sender),
			},
			Elements: FromBaseMessage(msg.Elements),
		}))
	case message.GroupMessage:
		b.elements = append(b.elements, lgrMessage.NewGroupReply(&lgrMessage.GroupMessage{
			ID:   uint32(id),
			Time: uint32(msg.Time.Unix()),
			Sender: &lgrMessage.Sender{
				Uin: uint32(sender),
			},
			Elements: FromBaseMessage(msg.Elements),
		}))
	}
	return b
}

func (b *messageBuilder) text(text string) *messageBuilder {
	length := len(b.elements)
	if length > 0 && b.elements[length-1].Type() == lgrMessage.Text {
		b.elements[length-1].(*lgrMessage.TextElement).Content += text
	} else {
		b.elements = append(b.elements, lgrMessage.NewText(text))
	}
	return b
}

func (b *messageBuilder) image(path string) *messageBuilder {
	elem, err := lgrMessage.NewFileImage(path)
	if err == nil {
		b.elements = append(b.elements, elem)
	}
	return b
}

func (b *messageBuilder) mention(strUin string) *messageBuilder {
	uin, err := strconv.ParseUint(strUin, 10, 64)
	if err != nil {
		return b
	}
	b.elements = append(b.elements, lgrMessage.NewAt(uint32(uin)))
	return b
}

func (b *messageBuilder) video(path string) *messageBuilder {
	elem, err := lgrMessage.NewFileVideo(path, nil)
	if err == nil {
		b.elements = append(b.elements, elem)
	}
	return b
}

func (b *messageBuilder) file(path string, name ...string) *messageBuilder {
	if len(name) > 0 && name[0] == "" {
		name = nil
	}
	elem, err := lgrMessage.NewLocalFile(path, name...)
	if err == nil {
		b.elements = append(b.elements, elem)
	}
	return b
}

func (b *messageBuilder) voice(path string) *messageBuilder {
	elem, err := lgrMessage.NewFileRecord(path)
	if err == nil {
		b.elements = append(b.elements, elem)
	}
	return b
}

func (b *messageBuilder) sticker(sid string) *messageBuilder {
	id, err := strconv.ParseUint(sid, 10, 16)
	if err == nil {
		b.elements = append(b.elements, lgrMessage.NewFace(uint16(id)))
	}
	return b
}
