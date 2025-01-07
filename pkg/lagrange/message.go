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
			b.Append(
				message.Voice,
				"[录音]",
				fmt.Sprintf("%x", elem.Md5),
			)
		case *LgrMessage.ImageElement:
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
			b.Append(message.Video,
				"[视频]",
				fmt.Sprintf("%x", elem.Md5),
			)
		}
	}
	return b.Build()
}

func FromBaseMessage(msg []*message.Element) []LgrMessage.IMessageElement {
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
	elements []LgrMessage.IMessageElement
}

func (b *messageBuilder) Build() []LgrMessage.IMessageElement {
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
		b.elements = append(b.elements, LgrMessage.NewPrivateReply(&LgrMessage.PrivateMessage{
			ID:   uint32(id),
			Time: uint32(msg.Time.Unix()),
			Sender: &LgrMessage.Sender{
				Uin: uint32(sender),
			},
			Elements: FromBaseMessage(msg.Elements),
		}))
	case message.GroupMessage:
		b.elements = append(b.elements, LgrMessage.NewGroupReply(&LgrMessage.GroupMessage{
			ID:   uint32(id),
			Time: uint32(msg.Time.Unix()),
			Sender: &LgrMessage.Sender{
				Uin: uint32(sender),
			},
			Elements: FromBaseMessage(msg.Elements),
		}))
	}
	return b
}

func (b *messageBuilder) text(text string) *messageBuilder {
	length := len(b.elements)
	if length > 0 && b.elements[length-1].Type() == LgrMessage.Text {
		b.elements[length-1].(*LgrMessage.TextElement).Content += text
	} else {
		b.elements = append(b.elements, LgrMessage.NewText(text))
	}
	return b
}

func (b *messageBuilder) image(path string) *messageBuilder {
	elem, err := LgrMessage.NewFileImage(path)
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
	b.elements = append(b.elements, LgrMessage.NewAt(uint32(uin)))
	return b
}

func (b *messageBuilder) video(path string) *messageBuilder {
	elem, err := LgrMessage.NewFileVideo(path, nil)
	if err == nil {
		b.elements = append(b.elements, elem)
	}
	return b
}

func (b *messageBuilder) file(path string, name ...string) *messageBuilder {
	if len(name) > 0 && name[0] == "" {
		name = nil
	}
	elem, err := LgrMessage.NewLocalFile(path, name...)
	if err == nil {
		b.elements = append(b.elements, elem)
	}
	return b
}

func (b *messageBuilder) voice(path string) *messageBuilder {
	elem, err := LgrMessage.NewFileRecord(path)
	if err == nil {
		b.elements = append(b.elements, elem)
	}
	return b
}

func (b *messageBuilder) sticker(sid string) *messageBuilder {
	id, err := strconv.ParseUint(sid, 10, 16)
	if err == nil {
		b.elements = append(b.elements, LgrMessage.NewFace(uint16(id)))
	}
	return b
}
