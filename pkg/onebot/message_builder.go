package onebot

import (
	"fmt"
	"strconv"
	"strings"

	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
)

type MessageBuilder struct {
	service  *Service
	elements []*botc.MessageElement
}

func (mb *MessageBuilder) Protocol() string {
	return "onebot"
}

func (mb *MessageBuilder) Text(text string) botc.MessageBuilder {
	mb.elements = append(mb.elements, &botc.MessageElement{
		Type:    botc.TextElement,
		Content: text,
	})
	return mb
}

func (mb *MessageBuilder) Quote(msg *botc.BaseMessage) botc.MessageBuilder {
	mb.elements = append(mb.elements, &botc.MessageElement{
		Type:    botc.QuoteElement,
		Content: msg.ID,
	})
	return mb
}

func (mb *MessageBuilder) Mention(id string) botc.MessageBuilder {
	mb.elements = append(mb.elements, &botc.MessageElement{
		Type:    botc.MentionElement,
		Content: id,
	})
	return mb
}

func (mb *MessageBuilder) ImageFromFile(path string) botc.MessageBuilder {
	// For OneBot, we need to upload the file first or use the file path directly
	mb.elements = append(mb.elements, &botc.MessageElement{
		Type:   botc.ImageElement,
		Source: fmt.Sprintf("file://%s", path),
	})
	return mb
}

func (mb *MessageBuilder) ImageFromUrl(url string) botc.MessageBuilder {
	mb.elements = append(mb.elements, &botc.MessageElement{
		Type:   botc.ImageElement,
		Source: url,
	})
	return mb
}

func (mb *MessageBuilder) ImageFromData(data []byte) botc.MessageBuilder {
	// For OneBot, we could encode as base64 or save to temp file
	// Using base64 encoding as per OneBot specification
	encoded := fmt.Sprintf("base64://%s", encodeBase64(data))
	mb.elements = append(mb.elements, &botc.MessageElement{
		Type:   botc.ImageElement,
		Source: encoded,
	})
	return mb
}

func (mb *MessageBuilder) ReplyTo(ctx botc.MessageContext) (*botc.BaseMessage, error) {
	// Get the message context and reply appropriately
	msg := ctx.Message()

	if msg.MessageType == botc.DirectMessage && msg.Sender != nil && msg.Sender.User != nil {
		target := entity.User{
			Base: &entity.Base{
				ID:   msg.Sender.User.ID,
				Name: msg.Sender.User.Name,
			},
		}
		return mb.service.getContext().SendDirectMessage(target, mb.elements)
	} else if msg.MessageType == botc.GroupMessage && msg.Sender != nil && msg.Sender.From != nil {
		target := entity.Group{
			Base: &entity.Base{
				ID:   msg.Sender.From.ID,
				Name: msg.Sender.From.Name,
			},
		}
		return mb.service.getContext().SendGroupMessage(target, mb.elements)
	}

	return nil, fmt.Errorf("unable to determine reply target")
}

func (mb *MessageBuilder) Send(id string) (*botc.BaseMessage, error) {
	// Parse ID to determine if it's a user or group
	if strings.HasPrefix(id, "onebot:") {
		// Remove prefix for parsing
		id = strings.TrimPrefix(id, "onebot:")
	}

	// Try to parse as user ID first
	if userID, err := strconv.ParseInt(id, 10, 64); err == nil {
		target := entity.User{
			Base: &entity.Base{
				ID: genUserID(userID),
			},
		}
		return mb.service.getContext().SendDirectMessage(target, mb.elements)
	}

	return nil, fmt.Errorf("invalid target ID: %s", id)
}

// Additional helper methods for OneBot-specific functionality

func (mb *MessageBuilder) Voice(url string) botc.MessageBuilder {
	mb.elements = append(mb.elements, &botc.MessageElement{
		Type:   botc.VoiceElement,
		Source: url,
	})
	return mb
}

func (mb *MessageBuilder) Video(url string) botc.MessageBuilder {
	mb.elements = append(mb.elements, &botc.MessageElement{
		Type:   botc.VideoElement,
		Source: url,
	})
	return mb
}

func (mb *MessageBuilder) At(target string) botc.MessageBuilder {
	return mb.Mention(target)
}

func (mb *MessageBuilder) Face(id string) botc.MessageBuilder {
	mb.elements = append(mb.elements, &botc.MessageElement{
		Type:    botc.StickerElement,
		Content: id,
	})
	return mb
}

func (mb *MessageBuilder) Reply(messageID string) botc.MessageBuilder {
	mb.elements = append(mb.elements, &botc.MessageElement{
		Type:    botc.QuoteElement,
		Content: messageID,
	})
	return mb
}

func (mb *MessageBuilder) File(url string) botc.MessageBuilder {
	mb.elements = append(mb.elements, &botc.MessageElement{
		Type:   botc.FileElement,
		Source: url,
	})
	return mb
}

func (mb *MessageBuilder) Elements() []*botc.MessageElement {
	return mb.elements
}

// Utility functions for parsing IDs and handling CQ codes

func encodeBase64(data []byte) string {
	// Simple base64 encoding implementation
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var result strings.Builder

	for i := 0; i < len(data); i += 3 {
		var chunk uint32
		switch {
		case i+2 < len(data):
			chunk = uint32(data[i])<<16 | uint32(data[i+1])<<8 | uint32(data[i+2])
			result.WriteByte(chars[chunk>>18])
			result.WriteByte(chars[(chunk>>12)&63])
			result.WriteByte(chars[(chunk>>6)&63])
			result.WriteByte(chars[chunk&63])
		case i+1 < len(data):
			chunk = uint32(data[i])<<16 | uint32(data[i+1])<<8
			result.WriteByte(chars[chunk>>18])
			result.WriteByte(chars[(chunk>>12)&63])
			result.WriteByte(chars[(chunk>>6)&63])
			result.WriteByte('=')
		default:
			chunk = uint32(data[i]) << 16
			result.WriteByte(chars[chunk>>18])
			result.WriteByte(chars[(chunk>>12)&63])
			result.WriteString("==")
		}
	}

	return result.String()
}

func parseUserID(id string) (int64, error) {
	// Remove any prefix if present
	id = strings.TrimPrefix(id, "onebot:")
	return strconv.ParseInt(id, 10, 64)
}

func parseGroupID(id string) (int64, error) {
	// Remove any prefix if present
	id = strings.TrimPrefix(id, "onebot:")
	return strconv.ParseInt(id, 10, 64)
}

func genUserID(uin int64) string {
	return fmt.Sprintf("onebot:%d", uin)
}

func genGroupID(uin int64) string {
	return fmt.Sprintf("onebot:%d", uin)
}

func extractTextContent(elements []*botc.MessageElement) string {
	var texts []string
	for _, elem := range elements {
		if elem.Type == botc.TextElement {
			texts = append(texts, elem.Content)
		}
	}
	return strings.Join(texts, "")
}

// CQ code escaping functions
func escapeCQCode(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "[", "&#91;")
	text = strings.ReplaceAll(text, "]", "&#93;")
	return text
}

func unescapeCQCode(text string) string {
	text = strings.ReplaceAll(text, "&#93;", "]")
	text = strings.ReplaceAll(text, "&#91;", "[")
	text = strings.ReplaceAll(text, "&amp;", "&")
	return text
}

func parseCQCode(cqCode string) (string, map[string]string) {
	// Parse CQ code format: [CQ:type,param1=value1,param2=value2]
	if !strings.HasPrefix(cqCode, "[CQ:") || !strings.HasSuffix(cqCode, "]") {
		return "", nil
	}

	content := cqCode[4 : len(cqCode)-1] // Remove [CQ: and ]
	parts := strings.Split(content, ",")
	if len(parts) == 0 {
		return "", nil
	}

	cqType := parts[0]
	params := make(map[string]string)

	for i := 1; i < len(parts); i++ {
		if kv := strings.SplitN(parts[i], "=", 2); len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			// Unescape CQ code parameters
			value = strings.ReplaceAll(value, "&#44;", ",")
			value = strings.ReplaceAll(value, "&#93;", "]")
			value = strings.ReplaceAll(value, "&#91;", "[")
			value = strings.ReplaceAll(value, "&amp;", "&")
			params[key] = value
		}
	}

	return cqType, params
}
