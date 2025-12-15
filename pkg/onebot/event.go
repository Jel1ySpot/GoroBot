package onebot

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/Jel1ySpot/GoroBot/pkg/util"
)

// OneBot event structures
type BaseEvent struct {
	Time     int64  `json:"time"`
	SelfID   int64  `json:"self_id"`
	PostType string `json:"post_type"`
}

type MessageEvent struct {
	BaseEvent
	MessageType string          `json:"message_type"`
	SubType     string          `json:"sub_type"`
	MessageID   int64           `json:"message_id"`
	UserID      int64           `json:"user_id"`
	GroupID     int64           `json:"group_id,omitempty"`
	Message     json.RawMessage `json:"message"`
	RawMessage  string          `json:"raw_message"`
	Font        int32           `json:"font"`
	Sender      Sender          `json:"sender"`
	Anonymous   *Anonymous      `json:"anonymous,omitempty"`
}

type Sender struct {
	UserID   int64  `json:"user_id"`
	Nickname string `json:"nickname"`
	Card     string `json:"card,omitempty"`
	Sex      string `json:"sex"`
	Age      int32  `json:"age"`
	Area     string `json:"area,omitempty"`
	Level    string `json:"level,omitempty"`
	Role     string `json:"role,omitempty"`
	Title    string `json:"title,omitempty"`
}

type Anonymous struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Flag string `json:"flag"`
}

type NoticeEvent struct {
	BaseEvent
	NoticeType string `json:"notice_type"`
	SubType    string `json:"sub_type,omitempty"`
	GroupID    int64  `json:"group_id,omitempty"`
	UserID     int64  `json:"user_id,omitempty"`
	OperatorID int64  `json:"operator_id,omitempty"`
	TargetID   int64  `json:"target_id,omitempty"`
	Duration   int64  `json:"duration,omitempty"`
	MessageID  int64  `json:"message_id,omitempty"`
	File       *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Size  int64  `json:"size"`
		BusID int64  `json:"busid"`
	} `json:"file,omitempty"`
}

type RequestEvent struct {
	BaseEvent
	RequestType string `json:"request_type"`
	SubType     string `json:"sub_type,omitempty"`
	UserID      int64  `json:"user_id"`
	GroupID     int64  `json:"group_id,omitempty"`
	Comment     string `json:"comment"`
	Flag        string `json:"flag"`
}

type MetaEvent struct {
	BaseEvent
	MetaEventType string      `json:"meta_event_type"`
	SubType       string      `json:"sub_type,omitempty"`
	Status        interface{} `json:"status,omitempty"`
	Interval      int64       `json:"interval,omitempty"`
}

// Message context for OneBot
type MessageContext struct {
	service      *Service
	messageEvent *MessageEvent
	message      *botc.BaseMessage
}

func (mc *MessageContext) Protocol() string {
	return "onebot"
}

func (mc *MessageContext) BotContext() botc.BotContext {
	return mc.service.getContext()
}

func (mc *MessageContext) String() string {
	return mc.message.Content
}

func (mc *MessageContext) Message() *botc.BaseMessage {
	return mc.message
}

func (mc *MessageContext) SenderID() string {
	return genUserID(mc.messageEvent.UserID)
}

func (mc *MessageContext) NewMessageBuilder() botc.MessageBuilder {
	return &MessageBuilder{
		service:  mc.service,
		elements: make([]*botc.MessageElement, 0),
	}
}

func (mc *MessageContext) ReplyText(a ...any) (*botc.BaseMessage, error) {
	// Convert arguments to string
	text := fmt.Sprint(a...)
	builder := mc.NewMessageBuilder().(*MessageBuilder)
	builder.Text(text)
	return mc.Reply(builder.Elements())
}

func (mc *MessageContext) Reply(elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	ctx := mc.service.getContext()

	if mc.messageEvent.MessageType == "private" {
		target := entity.User{
			Base: &entity.Base{
				ID:   genUserID(mc.messageEvent.UserID),
				Name: mc.messageEvent.Sender.Nickname,
			},
		}
		return ctx.SendDirectMessage(target, elements)
	} else if mc.messageEvent.MessageType == "group" {
		// Try to get group name from cache
		var groupName string
		if group, exists := mc.service.getCachedGroupInfo(mc.messageEvent.GroupID); exists {
			groupName = group.GroupName
		} else {
			groupName = fmt.Sprintf("Group %d", mc.messageEvent.GroupID)
		}

		target := entity.Group{
			Base: &entity.Base{
				ID:   genGroupID(mc.messageEvent.GroupID),
				Name: groupName,
			},
		}
		return ctx.SendGroupMessage(target, elements)
	}

	return nil, fmt.Errorf("unsupported message type: %s", mc.messageEvent.MessageType)
}

// Event handling registration
func (s *Service) registerEventHandlers() {
	// This method sets up the event handlers for different connection modes
	// The actual event processing is handled in the connection-specific methods
}

// Parse OneBot message to GoroBot format
func (s *Service) parseMessage(messageEvent *MessageEvent) (*botc.BaseMessage, error) {
	var elements []*botc.MessageElement
	var content string

	// Parse message based on format
	if s.config.MessageFormat == "array" {
		var segments []map[string]interface{}
		if err := json.Unmarshal(messageEvent.Message, &segments); err == nil {
			elements, content = s.parseMessageSegments(segments)
		}
	} else {
		// String format with CQ codes
		var messageStr string
		if err := json.Unmarshal(messageEvent.Message, &messageStr); err == nil {
			elements, content = s.parseCQCodeMessage(messageStr)
		}
	}

	// Create sender entity
	sender := &entity.Sender{
		User: &entity.User{
			Base: &entity.Base{
				ID:   genUserID(messageEvent.UserID),
				Name: messageEvent.Sender.Nickname,
			},
			Nickname: util.CoalesceString(messageEvent.Sender.Card, messageEvent.Sender.Nickname),
			Age:      uint32(messageEvent.Sender.Age),
			Authority: func() entity.Authority {
				switch messageEvent.Sender.Role {
				case "owner":
					return entity.GroupOwner
				case "admin":
					return entity.GroupAdmin
				default:
					return entity.Member
				}
			}(),
		},
	}

	// Create message
	message := &botc.BaseMessage{
		ID:       fmt.Sprintf("%d", messageEvent.MessageID),
		Content:  content,
		Elements: elements,
		Sender:   sender,
		Time:     time.Unix(messageEvent.Time, 0),
	}

	// Set message type and target
	if messageEvent.MessageType == "private" {
		message.MessageType = botc.DirectMessage
	} else if messageEvent.MessageType == "group" {
		message.MessageType = botc.GroupMessage
		// Try to get group name from cache
		var groupName string
		if group, exists := s.getCachedGroupInfo(messageEvent.GroupID); exists {
			groupName = group.GroupName
		} else {
			groupName = fmt.Sprintf("Group %d", messageEvent.GroupID)
		}

		// Set the group information in the sender's From field
		sender.From = &entity.Base{
			ID:   genGroupID(messageEvent.GroupID),
			Name: groupName,
		}
	}

	return message, nil
}

func (s *Service) parseMessageSegments(segments []map[string]interface{}) ([]*botc.MessageElement, string) {
	b := botc.NewBuilder()

	for _, segment := range segments {
		segType, ok := segment["type"].(string)
		if !ok {
			continue
		}

		data, ok := segment["data"].(map[string]interface{})
		if !ok {
			data = make(map[string]interface{})
		}

		switch segType {
		case "text":
			if text, ok := data["text"].(string); ok {
				b.Append(botc.TextElement, text, "")
			}
		case "image":
			if file, ok := data["file"].(string); ok {
				url, _ := data["url"].(string)
				if url == "" {
					url = file
				}

				// Save resource and get resource ID
				resourceID := s.saveImageResource(url)

				b.Append(botc.ImageElement, util.CoalesceString(data["summary"], "[图片]"), resourceID)
			}
		case "record":
			if file, ok := data["file"].(string); ok {
				url, _ := data["url"].(string)
				if url == "" {
					url = file
				}

				// Save resource and get resource ID
				resourceID := s.saveVoiceResource(url)
				b.Append(botc.VoiceElement, "[语音]", resourceID)
			}
		case "video":
			if file, ok := data["file"].(string); ok {
				url, _ := data["url"].(string)
				if url == "" {
					url = file
				}

				// Save resource and get resource ID
				resourceID := s.saveVideoResource(url)
				b.Append(botc.VideoElement, "[视频]", resourceID)
			}
		case "at":
			if qq, ok := data["qq"].(string); ok {
				b.Append(botc.MentionElement, fmt.Sprintf("@%s", qq), qq)
			}
		case "face":
			if id, ok := data["id"].(string); ok {
				b.Append(botc.StickerElement, "[表情]", id)
			}
		case "reply":
			if id, ok := data["id"].(string); ok {
				b.Append(botc.QuoteElement, "[回复]", id)
			}
		}
	}

	elements := b.Build()
	return elements, botc.ElemsToString(elements)
}

func (s *Service) parseCQCodeMessage(message string) ([]*botc.MessageElement, string) {
	b := botc.NewBuilder()

	// Split message by CQ codes
	parts := strings.Split(message, "[CQ:")

	// First part is always text (might be empty)
	if len(parts) > 0 && parts[0] != "" {
		text := unescapeCQCode(parts[0])
		b.Append(botc.TextElement, text, "")
	}

	// Process CQ codes
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		cqEnd := strings.Index(part, "]")
		if cqEnd == -1 {
			continue
		}

		cqCode := "[CQ:" + part[:cqEnd+1]
		remaining := part[cqEnd+1:]

		// Parse CQ code
		cqType, params := parseCQCode(cqCode)
		switch cqType {
		case "image":
			if file, ok := params["file"]; ok {
				// Save resource and get resource ID
				resourceID := s.saveImageResource(file)
				b.Append(botc.ImageElement, "[图片]", resourceID)
			}
		case "record":
			if file, ok := params["file"]; ok {
				// Save resource and get resource ID
				resourceID := s.saveVoiceResource(file)
				b.Append(botc.VoiceElement, "[语音]", resourceID)
			}
		case "video":
			if file, ok := params["file"]; ok {
				// Save resource and get resource ID
				resourceID := s.saveVideoResource(file)
				b.Append(botc.VideoElement, "[视频]", resourceID)
			}
		case "at":
			if qq, ok := params["qq"]; ok {
				b.Append(botc.MentionElement, fmt.Sprintf("@%s", qq), qq)
			}
		case "face":
			if id, ok := params["id"]; ok {
				b.Append(botc.StickerElement, "[表情]", id)
			}
		case "reply":
			if id, ok := params["id"]; ok {
				b.Append(botc.QuoteElement, "[回复]", id)
			}
		}

		// Add remaining text
		if remaining != "" {
			text := unescapeCQCode(remaining)
			b.Append(botc.TextElement, text, "")
		}
	}

	elements := b.Build()
	return elements, botc.ElemsToString(elements)
}

// Process incoming OneBot events
func (s *Service) processEvent(eventData []byte) error {
	var baseEvent BaseEvent
	if err := json.Unmarshal(eventData, &baseEvent); err != nil {
		return fmt.Errorf("failed to parse base event: %v", err)
	}

	switch baseEvent.PostType {
	case "message":
		return s.processMessageEvent(eventData)
	case "notice":
		return s.processNoticeEvent(eventData)
	case "request":
		return s.processRequestEvent(eventData)
	case "meta_event":
		return s.processMetaEvent(eventData)
	default:
		s.logger.Debug("Unknown event type: %s", baseEvent.PostType)
		return nil
	}
}

func (s *Service) processMessageEvent(eventData []byte) error {
	var messageEvent MessageEvent
	if err := json.Unmarshal(eventData, &messageEvent); err != nil {
		return fmt.Errorf("failed to parse message event: %v", err)
	}

	s.logger.Debug("Processing message event from user %d (type: %s)", messageEvent.UserID, messageEvent.MessageType)

	// Skip self messages if configured
	if s.config.IgnoreSelf && messageEvent.UserID == messageEvent.SelfID {
		s.logger.Debug("Ignoring self message from user %d", messageEvent.UserID)
		return nil
	}

	// Parse message
	message, err := s.parseMessage(&messageEvent)
	if err != nil {
		s.logger.Error("Failed to parse message from user %d: %v", messageEvent.UserID, err)
		return fmt.Errorf("failed to parse message: %v", err)
	}

	// Create message context
	messageCtx := &MessageContext{
		service:      s,
		messageEvent: &messageEvent,
		message:      message,
	}

	s.logger.Debug("Triggering message event for content: %s", message.Content)

	// Check if message is a command based on configured prefix
	if s.config.CommandPrefix != "" && strings.HasPrefix(message.Content, s.config.CommandPrefix) {
		// Extract command text without prefix
		commandText := strings.TrimSpace(message.Content[len(s.config.CommandPrefix):])
		if commandText != "" {
			s.logger.Debug("Command detected: %s", commandText)

			// Emit command event
			s.grb.CommandEmit(
				command.NewCommandContext(messageCtx, commandText),
			)
			return nil
		}
	}

	// Trigger regular message event
	if err := s.grb.EventEmit("message", messageCtx); err != nil {
		s.logger.Error("Failed to emit message event: %v", err)
		return fmt.Errorf("failed to emit message event: %v", err)
	}

	return nil
}

func (s *Service) processNoticeEvent(eventData []byte) error {
	var noticeEvent NoticeEvent
	if err := json.Unmarshal(eventData, &noticeEvent); err != nil {
		return fmt.Errorf("failed to parse notice event: %v", err)
	}

	s.logger.Debug("Received notice event: %s/%s", noticeEvent.NoticeType, noticeEvent.SubType)

	// Handle cache invalidation based on notice type
	switch noticeEvent.NoticeType {
	case "friend_add":
		// Friend added, invalidate friend cache
		s.logger.Debug("Friend added, invalidating friend cache")
		s.invalidateFriendCache()
	case "group_increase", "group_decrease":
		// Group member changes, invalidate group cache
		s.logger.Debug("Group member change detected, invalidating group cache")
		s.invalidateGroupCache()
	case "group_admin":
		// Admin changes, invalidate group cache
		s.logger.Debug("Group admin change detected, invalidating group cache")
		s.invalidateGroupCache()
	}

	return nil
}

func (s *Service) processRequestEvent(eventData []byte) error {
	var requestEvent RequestEvent
	if err := json.Unmarshal(eventData, &requestEvent); err != nil {
		return fmt.Errorf("failed to parse request event: %v", err)
	}

	s.logger.Debug("Received request event: %s", requestEvent.RequestType)
	// Additional request event processing can be added here
	return nil
}

func (s *Service) processMetaEvent(eventData []byte) error {
	var metaEvent MetaEvent
	if err := json.Unmarshal(eventData, &metaEvent); err != nil {
		return fmt.Errorf("failed to parse meta event: %v", err)
	}

	s.logger.Debug("Received meta event: %s", metaEvent.MetaEventType)

	if metaEvent.MetaEventType == "heartbeat" {
		s.logger.Debug("Heartbeat received")
	}

	return nil
}

// Resource saving methods for different media types

func (s *Service) saveImageResource(url string) string {
	if url == "" {
		return ""
	}

	// Try to save the resource using the core resource system
	resource, err := s.grb.SaveRemoteResource(url)
	if err != nil {
		s.logger.Error("Failed to save image resource from %s: %v", url, err)
		return url // Return original URL as fallback
	}

	s.logger.Debug("Saved image resource: %s -> %s", url, resource.ID)
	return resource.ID
}

func (s *Service) saveVoiceResource(url string) string {
	if url == "" {
		return ""
	}

	// Try to save the resource using the core resource system
	resource, err := s.grb.SaveRemoteResource(url)
	if err != nil {
		s.logger.Error("Failed to save voice resource from %s: %v", url, err)
		return url // Return original URL as fallback
	}

	s.logger.Debug("Saved voice resource: %s -> %s", url, resource.ID)
	return resource.ID
}

func (s *Service) saveVideoResource(url string) string {
	if url == "" {
		return ""
	}

	// Try to save the resource using the core resource system
	resource, err := s.grb.SaveRemoteResource(url)
	if err != nil {
		s.logger.Error("Failed to save video resource from %s: %v", url, err)
		return url // Return original URL as fallback
	}

	s.logger.Debug("Saved video resource: %s -> %s", url, resource.ID)
	return resource.ID
}
