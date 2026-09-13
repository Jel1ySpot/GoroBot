package qbot

import (
	"context"
	"fmt"
	urlpkg "net/url"
	"os"
	"path"
	"strings"
	"time"

	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
)

type MessageBuilder struct {
	*MessageToCreate
	fromMsg   *MessageContext
	MediaData []byte
	MediaType uint64
	ctx       *Service
}

func NewMessageBuilder(from *MessageContext) *MessageBuilder {
	return &MessageBuilder{
		MessageToCreate: &MessageToCreate{},
		fromMsg:         from,
		ctx:             from.bot,
	}
}

func (m *MessageBuilder) Protocol() string {
	return "qbot"
}

func (m *MessageBuilder) Text(text string) botc.MessageBuilder {
	if m.Content != "" {
		m.Content += " "
	}
	m.Content += text
	return m
}

func (m *MessageBuilder) Quote(msg *botc.BaseMessage) botc.MessageBuilder {
	info, ok := entity.ParseInfo(msg.ID)
	if !ok || info.Protocol != m.Protocol() || info.Args[0] != "msg" {
		return m
	}
	msgID := ""
	if len(info.Args) >= 3 {
		msgID = info.Args[2]
	} else if len(info.Args) >= 2 {
		msgID = info.Args[1]
	}
	m.MessageReference = &MessageReference{
		MessageID:             msgID,
		IgnoreGetMessageError: true,
	}
	return m
}

func (m *MessageBuilder) Mention(id string) botc.MessageBuilder {
	info, ok := entity.ParseInfo(id)
	if ok && info.Protocol == m.Protocol() && info.Args[0] == "user" {
		m.Text(fmt.Sprintf("<qqbot-at-user id=\"%s\" /> ", info.Args[1]))
	} else if ok && info.Protocol == m.Protocol() && info.Args[0] == "everyone" {
		m.Text("<qqbot-at-everyone /> ")
	}
	return m
}

func (m *MessageBuilder) CmdEnter(text string) *MessageBuilder {
	m.Text(fmt.Sprintf("<qqbot-cmd-enter text=\"%s\" /> ", urlpkg.QueryEscape(text)))
	return m
}

func (m *MessageBuilder) CmdInput(text, show string, reference bool) *MessageBuilder {
	if show != "" {
		show = fmt.Sprintf("show=\"%s\" ", urlpkg.QueryEscape(show))
	}
	m.Text(fmt.Sprintf("<qqbot-cmd-input text=\"%s\" %sreference=\"%t\" /> ", urlpkg.QueryEscape(text), show, reference))
	return m
}

func (m *MessageBuilder) Emoji(id uint) *MessageBuilder {
	m.Text(fmt.Sprintf(`<emoji:%d>`, id))
	return m
}

func (m *MessageBuilder) ImageFromFile(path string) botc.MessageBuilder {
	data, err := os.ReadFile(path)
	if err != nil {
		return m
	}
	return m.ImageFromData(data)
}

func (m *MessageBuilder) ImageFromUrl(url string) botc.MessageBuilder {
	refLink := urlpkg.Values{
		"url": {url},
		"ext": {strings.TrimPrefix(path.Ext(url), ".")},
	}.Encode()
	id := m.ctx.grb.SaveResourceLink(m.ctx.ID(), refLink)
	if pathStr, err := m.ctx.grb.LoadResourceFromID(id); err == nil {
		if data, err := os.ReadFile(pathStr); err == nil {
			m.ImageFromData(data)
		}
	}
	return m
}

func (m *MessageBuilder) ImageFromData(data []byte) botc.MessageBuilder {
	m.MediaData = data
	m.MediaType = ImageFileType
	return m
}

func (m *MessageBuilder) VideoFromFile(path string) *MessageBuilder {
	data, err := os.ReadFile(path)
	if err != nil {
		return m
	}
	return m.VideoFromData(data)
}

func (m *MessageBuilder) VideoFromUrl(url string) *MessageBuilder {
	refLink := urlpkg.Values{
		"url": {url},
		"ext": {strings.TrimPrefix(path.Ext(url), ".")},
	}.Encode()
	id := m.ctx.grb.SaveResourceLink(m.ctx.ID(), refLink)
	if pathStr, err := m.ctx.grb.LoadResourceFromID(id); err == nil {
		if data, err := os.ReadFile(pathStr); err == nil {
			m.VideoFromData(data)
		}
	}
	return m
}

func (m *MessageBuilder) VideoFromData(data []byte) *MessageBuilder {
	m.MediaData = data
	m.MediaType = VideoFileType
	return m
}

func (m *MessageBuilder) VoiceFromFile(path string) *MessageBuilder {
	data, err := os.ReadFile(path)
	if err != nil {
		return m
	}
	return m.VoiceFromData(data)
}

func (m *MessageBuilder) VoiceFromUrl(url string) *MessageBuilder {
	refLink := urlpkg.Values{
		"url": {url},
		"ext": {strings.TrimPrefix(path.Ext(url), ".")},
	}.Encode()
	id := m.ctx.grb.SaveResourceLink(m.ctx.ID(), refLink)
	if pathStr, err := m.ctx.grb.LoadResourceFromID(id); err == nil {
		if data, err := os.ReadFile(pathStr); err == nil {
			m.VoiceFromData(data)
		}
	}
	return m
}

func (m *MessageBuilder) VoiceFromData(data []byte) *MessageBuilder {
	m.MediaData = data
	m.MediaType = VoiceFileType
	return m
}

// ApplyElements 将 MessageElement 列表应用到构建器
func (m *MessageBuilder) ApplyElements(elements []*botc.MessageElement) *MessageBuilder {
	for _, el := range elements {
		switch el.Type {
		case botc.TextElement:
			m.Text(el.Content)
		case botc.ImageElement:
			if pathStr, err := m.ctx.grb.LoadResourceFromID(el.Source); err == nil {
				m.ImageFromFile(pathStr)
			}
		case botc.VoiceElement:
			if pathStr, err := m.ctx.grb.LoadResourceFromID(el.Source); err == nil {
				if data, err := os.ReadFile(pathStr); err == nil {
					m.VoiceFromData(data)
				}
			}
		case botc.VideoElement:
			if pathStr, err := m.ctx.grb.LoadResourceFromID(el.Source); err == nil {
				if data, err := os.ReadFile(pathStr); err == nil {
					m.VideoFromData(data)
				}
			}
		case botc.MentionElement:
			m.Mention(el.Source)
		}
	}
	return m
}

func (m *MessageBuilder) Build() *MessageToCreate {
	m.MsgSeq = 1
	m.Timestamp = time.Now().Unix()
	if m.MessageToCreate.Media != nil {
		m.MsgType = 7
	}
	return m.MessageToCreate
}

func (m *MessageBuilder) ReplyTo(msg botc.MessageContext) (*botc.BaseMessage, error) {
	if msg.Protocol() != m.Protocol() {
		return nil, fmt.Errorf("expected message with %s protocol", msg.Protocol())
	}

	if ctx, ok := msg.(*command.Context); ok {
		msg = ctx.MessageContext
	}

	id := ""
	if sender := msg.Message().Sender; sender.From != nil {
		id = sender.From.ID
	} else {
		id = sender.ID
	}

	if err := m.prePostMedia(id); err != nil {
		m.ctx.logger.Warning("post media failed: %v", err)
	}

	return msg.(*MessageContext).reply(m.Build())
}

func (m *MessageBuilder) prePostMedia(id string) error {
	if m.MediaData == nil {
		return nil
	}
	if data, err := m.ctx.UploadFileData(id, m.MediaType, m.MediaData); err == nil {
		m.MessageToCreate.Media = &MediaInfo{
			FileInfo: data.FileInfo,
		}
		m.MessageToCreate.MsgType = 7
	} else {
		return err
	}
	return nil
}

func (m *MessageBuilder) Send(id string) (*botc.BaseMessage, error) {
	if err := m.prePostMedia(id); err != nil {
		m.ctx.logger.Warning("post media failed: %v", err)
	}

	info, ok := entity.ParseInfo(id)
	if !ok || info.Protocol != m.Protocol() {
		return nil, fmt.Errorf("invalid id %s", id)
	}
	idType, idVal := info.Args[0], info.Args[1]

	switch idType {
	case "user":
		data, err := m.ctx.api.PostC2CMessage(context.Background(), idVal, m.Build())
		if err != nil {
			return nil, err
		}
		return ParseMessage(m.ctx.grb, m.ctx, data), nil
	case "group":
		data, err := m.ctx.api.PostGroupMessage(context.Background(), idVal, m.Build())
		if err != nil {
			return nil, err
		}
		return ParseMessage(m.ctx.grb, m.ctx, data), nil
	case "channel":
		data, err := m.ctx.api.PostMessage(context.Background(), idVal, m.Build())
		if err != nil {
			return nil, err
		}
		return ParseMessage(m.ctx.grb, m.ctx, data), nil
	}
	return nil, fmt.Errorf("invalid id type %s", idType)
}
