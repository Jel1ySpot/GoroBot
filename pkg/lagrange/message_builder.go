package lagrange

import (
	"fmt"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	LgrMessage "github.com/LagrangeDev/LagrangeGo/message"
	"io"
	"net/http"
	"strconv"
)

type MessageBuilder struct {
	service  *Service
	elements []LgrMessage.IMessageElement
}

func (b *MessageBuilder) Protocol() string {
	return "lagrange"
}

func (b *MessageBuilder) Build() []LgrMessage.IMessageElement {
	return b.elements
}

func (b *MessageBuilder) Quote(msg *botc.BaseMessage) botc.MessageBuilder {
	id, ok := ParseUin(msg.ID)
	if !ok {
		return b
	}
	sender, ok := ParseUin(msg.Sender.ID)
	if !ok {
		return b
	}
	switch msg.MessageType {
	case botc.DirectMessage:
		b.elements = append(b.elements, LgrMessage.NewPrivateReply(&LgrMessage.PrivateMessage{
			ID:   id,
			Time: uint32(msg.Time.Unix()),
			Sender: &LgrMessage.Sender{
				Uin: sender,
			},
			Elements: TranslateMessageElement(b.service, msg.Elements),
		}))
	case botc.GroupMessage:
		b.elements = append(b.elements, LgrMessage.NewGroupReply(&LgrMessage.GroupMessage{
			ID:   id,
			Time: uint32(msg.Time.Unix()),
			Sender: &LgrMessage.Sender{
				Uin: sender,
			},
			Elements: TranslateMessageElement(b.service, msg.Elements),
		}))
	}
	return b
}

func (b *MessageBuilder) Text(text string) botc.MessageBuilder {
	length := len(b.elements)
	if length > 0 && b.elements[length-1].Type() == LgrMessage.Text {
		b.elements[length-1].(*LgrMessage.TextElement).Content += text
	} else {
		b.elements = append(b.elements, LgrMessage.NewText(text))
	}
	return b
}

func (b *MessageBuilder) ImageFromFile(path string) botc.MessageBuilder {
	elem, err := LgrMessage.NewFileImage(path)
	if err == nil {
		b.elements = append(b.elements, elem)
	}
	return b
}

func (b *MessageBuilder) ImageFromData(data []byte) botc.MessageBuilder {
	b.elements = append(b.elements, LgrMessage.NewImage(data))
	return b
}

func (b *MessageBuilder) ImageFromUrl(url string) botc.MessageBuilder {
	// 发送HTTP GET请求下载文件
	resp, err := http.Get(url)
	if err != nil {
		return b
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err == nil {
		b.elements = append(b.elements, LgrMessage.NewImage(data))
	}
	return b
}

func (b *MessageBuilder) Image(path string, data []byte, isSticker bool, summary ...string) botc.MessageBuilder {
	var (
		elem *LgrMessage.ImageElement
		err  error
	)
	if path != "" {
		elem, err = LgrMessage.NewFileImage(path, summary...)
	} else if data != nil {
		elem = LgrMessage.NewImage(data, summary...)
	} else {
		return b
	}
	if err != nil {
		return b
	}
	if isSticker {
		if summary == nil {
			elem.Summary = "[动画表情]"
		}
		elem.SubType = 7
	}
	b.elements = append(b.elements, elem)
	return b
}

func (b *MessageBuilder) Mention(id string) botc.MessageBuilder {
	uin, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return b
	}
	b.elements = append(b.elements, LgrMessage.NewAt(uint32(uin)))
	return b
}

func (b *MessageBuilder) VideoFromFile(path string) botc.MessageBuilder {
	elem, err := LgrMessage.NewFileVideo(path, nil)
	if err == nil {
		b.elements = append(b.elements, elem)
	}
	return b
}

func (b *MessageBuilder) File(path string, name ...string) botc.MessageBuilder {
	if len(name) > 0 && name[0] == "" {
		name = nil
	}
	elem, err := LgrMessage.NewLocalFile(path, name...)
	if err == nil {
		b.elements = append(b.elements, elem)
	}
	return b
}

func (b *MessageBuilder) Voice(path string) botc.MessageBuilder {
	elem, err := LgrMessage.NewFileRecord(path)
	if err == nil {
		b.elements = append(b.elements, elem)
	}
	return b
}

func (b *MessageBuilder) Sticker(sid string) botc.MessageBuilder {
	id, err := strconv.ParseUint(sid, 10, 16)
	if err == nil {
		b.elements = append(b.elements, LgrMessage.NewFace(uint16(id)))
	}
	return b
}

func (b *MessageBuilder) ReplyTo(msg botc.MessageContext) (*botc.BaseMessage, error) {
	if msg.Protocol() != "lagrange" {
		return nil, fmt.Errorf("expected protocol 'lagrange', got %s", msg.Protocol())
	}
	return msg.(*MessageContext).reply(b.elements)
}

func (b *MessageBuilder) Send(id string) (*botc.BaseMessage, error) {
	info, ok := entity.ParseInfo(id)
	if !ok || info.Protocol != "lagrange" {
		return nil, fmt.Errorf("invalid id %s", id)
	}
	idType := info.Args[0]
	uin, err := strconv.ParseUint(info.Args[1], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("cannot convert id %s to uin", info.Args[1])
	}

	client := b.service.qqClient

	switch idType {
	case "user":
		if event, err := client.SendPrivateMessage(uint32(uin), b.elements); err != nil {
			return nil, err
		} else {
			return ParseMessageEvent(b.service, event)
		}
	case "group":
		if event, err := client.SendGroupMessage(uint32(uin), b.elements); err != nil {
			return nil, err
		} else {
			return ParseMessageEvent(b.service, event)
		}
	}
	return nil, fmt.Errorf("unhandled id type %s", idType)
}
