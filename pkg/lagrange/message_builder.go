package lagrange

import (
	"fmt"
	GoroBot "github.com/Jel1ySpot/GoroBot/pkg/core"
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
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

func (b *MessageBuilder) Quote(msg *message.Base) GoroBot.MessageBuilder {
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
			Elements: b.service.FromBaseMessage(msg.Elements),
		}))
	case message.GroupMessage:
		b.elements = append(b.elements, LgrMessage.NewGroupReply(&LgrMessage.GroupMessage{
			ID:   uint32(id),
			Time: uint32(msg.Time.Unix()),
			Sender: &LgrMessage.Sender{
				Uin: uint32(sender),
			},
			Elements: b.service.FromBaseMessage(msg.Elements),
		}))
	}
	return b
}

func (b *MessageBuilder) Text(text string) GoroBot.MessageBuilder {
	length := len(b.elements)
	if length > 0 && b.elements[length-1].Type() == LgrMessage.Text {
		b.elements[length-1].(*LgrMessage.TextElement).Content += text
	} else {
		b.elements = append(b.elements, LgrMessage.NewText(text))
	}
	return b
}

func (b *MessageBuilder) ImageFromFile(path string) GoroBot.MessageBuilder {
	elem, err := LgrMessage.NewFileImage(path)
	if err == nil {
		b.elements = append(b.elements, elem)
	}
	return b
}

func (b *MessageBuilder) ImageFromData(data []byte) GoroBot.MessageBuilder {
	b.elements = append(b.elements, LgrMessage.NewImage(data))
	return b
}

func (b *MessageBuilder) ImageFromUrl(url string) GoroBot.MessageBuilder {
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

func (b *MessageBuilder) Mention(id string) GoroBot.MessageBuilder {
	uin, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return b
	}
	b.elements = append(b.elements, LgrMessage.NewAt(uint32(uin)))
	return b
}

func (b *MessageBuilder) VideoFromFile(path string) GoroBot.MessageBuilder {
	elem, err := LgrMessage.NewFileVideo(path, nil)
	if err == nil {
		b.elements = append(b.elements, elem)
	}
	return b
}

func (b *MessageBuilder) File(path string, name ...string) GoroBot.MessageBuilder {
	if len(name) > 0 && name[0] == "" {
		name = nil
	}
	elem, err := LgrMessage.NewLocalFile(path, name...)
	if err == nil {
		b.elements = append(b.elements, elem)
	}
	return b
}

func (b *MessageBuilder) Voice(path string) GoroBot.MessageBuilder {
	elem, err := LgrMessage.NewFileRecord(path)
	if err == nil {
		b.elements = append(b.elements, elem)
	}
	return b
}

func (b *MessageBuilder) Sticker(sid string) GoroBot.MessageBuilder {
	id, err := strconv.ParseUint(sid, 10, 16)
	if err == nil {
		b.elements = append(b.elements, LgrMessage.NewFace(uint16(id)))
	}
	return b
}

func (b *MessageBuilder) ReplyTo(msg message.Context) error {
	if msg.Protocol() != "lagrange" {
		return fmt.Errorf("expected protocol 'lagrange', got %s", msg.Protocol())
	}
	return msg.(*MessageContext).reply(b.elements)
}

func (b *MessageBuilder) Send(ctx GoroBot.BotContext, messageType message.Type, id string) error {
	if ctx.Protocol() != "lagrange" {
		return fmt.Errorf("expected protocol 'lagrange', got %s", ctx.Protocol())
	}

	uin, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return fmt.Errorf("cannot convert id %s to uin", id)
	}

	client := ctx.(*Context).service.qqClient

	switch messageType {
	case message.DirectMessage:
		if _, err := client.SendPrivateMessage(uint32(uin), b.elements); err != nil {
			return err
		}
	case message.GroupMessage:
		if _, err := client.SendGroupMessage(uint32(uin), b.elements); err != nil {
			return err
		}
	}
	return nil
}
