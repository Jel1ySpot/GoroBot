package onebot

import (
	"fmt"

	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
)

type Context struct {
	service *Service
}

func (ctx *Context) ID() string {
	return ctx.service.selfID
}

func (ctx *Context) Name() string {
	return ctx.service.nickname
}

func (ctx *Context) Protocol() string {
	return "onebot"
}

func (ctx *Context) Status() botc.LoginStatus {
	return ctx.service.status
}

func (ctx *Context) NewMessageBuilder() botc.MessageBuilder {
	return &MessageBuilder{
		service:  ctx.service,
		elements: make([]*botc.MessageElement, 0),
	}
}

func (ctx *Context) SendDirectMessage(target entity.User, elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	userID, err := parseUserID(target.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID %s: %v", target.ID, err)
	}

	message := ctx.translateToOneBot(elements)

	response, err := ctx.service.sendPrivateMessage(userID, message)
	if err != nil {
		return nil, err
	}

	return &botc.BaseMessage{
		ID:          fmt.Sprintf("%d", response.MessageID),
		MessageType: botc.DirectMessage,
		Content:     extractTextContent(elements),
		Elements:    elements,
		Sender: &entity.Sender{
			User: &entity.User{
				Base: &entity.Base{
					ID:   ctx.service.selfID,
					Name: ctx.service.nickname,
				},
			},
		},
	}, nil
}

func (ctx *Context) SendGroupMessage(target entity.Group, elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	groupID, err := parseGroupID(target.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid group ID %s: %v", target.ID, err)
	}

	message := ctx.translateToOneBot(elements)

	response, err := ctx.service.sendGroupMessage(groupID, message)
	if err != nil {
		return nil, err
	}

	return &botc.BaseMessage{
		ID:          fmt.Sprintf("%d", response.MessageID),
		MessageType: botc.GroupMessage,
		Content:     extractTextContent(elements),
		Elements:    elements,
		Sender: &entity.Sender{
			User: &entity.User{
				Base: &entity.Base{
					ID:   ctx.service.selfID,
					Name: ctx.service.nickname,
				},
			},
			From: target.Base,
		},
	}, nil
}

func (ctx *Context) GetMessageFileUrl(msg *botc.BaseMessage) (string, error) {
	// Find file element in message
	for _, elem := range msg.Elements {
		if elem.Type == botc.ImageElement || elem.Type == botc.VoiceElement || elem.Type == botc.FileElement {
			// For OneBot, the URL should be in the Source field
			if elem.Source != "" {
				return elem.Source, nil
			}
		}
	}
	return "", fmt.Errorf("no file element found in message")
}

func (ctx *Context) Contacts() []entity.User {
	friends, err := ctx.service.getFriendList()
	if err != nil {
		ctx.service.logger.Error("Failed to get friend list: %v", err)
		return nil
	}

	users := make([]entity.User, len(friends))
	for i, friend := range friends {
		users[i] = entity.User{
			Base: &entity.Base{
				ID:   fmt.Sprintf("%d", friend.UserID),
				Name: friend.Nickname,
			},
			Nickname:  friend.Remark,
			Authority: entity.Member,
		}
	}

	return users
}

func (ctx *Context) Groups() []entity.Group {
	groups, err := ctx.service.getGroupList()
	if err != nil {
		ctx.service.logger.Error("Failed to get group list: %v", err)
		return nil
	}

	result := make([]entity.Group, len(groups))
	for i, group := range groups {
		result[i] = entity.Group{
			Base: &entity.Base{
				ID:   fmt.Sprintf("%d", group.GroupID),
				Name: group.GroupName,
			},
			Members: nil, // Would need separate API call to get members
		}
	}

	return result
}

// GetGroupMemberInfo retrieves detailed information about a specific group member
func (ctx *Context) GetGroupMemberInfo(groupID, userID int64, noCache bool) (*GroupMember, error) {
	return ctx.service.getGroupMemberInfo(groupID, userID, noCache)
}

// RefreshCache forces a refresh of all cached data
func (ctx *Context) RefreshCache() error {
	return ctx.service.forceCacheRefresh()
}

// InvalidateCache clears all cached data
func (ctx *Context) InvalidateCache() {
	ctx.service.invalidateCache()
}

// InvalidateFriendCache clears only friend list cache
func (ctx *Context) InvalidateFriendCache() {
	ctx.service.invalidateFriendCache()
}

// InvalidateGroupCache clears only group list cache
func (ctx *Context) InvalidateGroupCache() {
	ctx.service.invalidateGroupCache()
}

// translateToOneBot converts GoroBot message elements to OneBot message format
func (ctx *Context) translateToOneBot(elements []*botc.MessageElement) interface{} {
	if ctx.service.config.MessageFormat == "string" {
		return ctx.elementsToString(elements)
	}
	return ctx.elementsToArray(elements)
}

func (ctx *Context) elementsToString(elements []*botc.MessageElement) string {
	var result string
	for _, elem := range elements {
		switch elem.Type {
		case botc.TextElement:
			result += escapeCQCode(elem.Content)
		case botc.ImageElement:
			// Get resource file path from resource ID
			if resource, err := ctx.service.grb.GetResource(elem.Source); err == nil {
				result += fmt.Sprintf("[CQ:image,file=file://%s]", resource.FilePath)
			} else {
				// Fallback to source as URL
				result += fmt.Sprintf("[CQ:image,file=%s]", elem.Source)
			}
		case botc.VoiceElement:
			// Get resource file path from resource ID
			if resource, err := ctx.service.grb.GetResource(elem.Source); err == nil {
				result += fmt.Sprintf("[CQ:record,file=file://%s]", resource.FilePath)
			} else {
				// Fallback to source as URL
				result += fmt.Sprintf("[CQ:record,file=%s]", elem.Source)
			}
		case botc.VideoElement:
			// Get resource file path from resource ID
			if resource, err := ctx.service.grb.GetResource(elem.Source); err == nil {
				result += fmt.Sprintf("[CQ:video,file=file://%s]", resource.FilePath)
			} else {
				// Fallback to source as URL
				result += fmt.Sprintf("[CQ:video,file=%s]", elem.Source)
			}
		case botc.MentionElement:
			result += fmt.Sprintf("[CQ:at,qq=%s]", elem.Source)
		case botc.StickerElement:
			result += fmt.Sprintf("[CQ:face,id=%s]", elem.Source)
		case botc.QuoteElement:
			result += fmt.Sprintf("[CQ:reply,id=%s]", elem.Source)
		}
	}
	return result
}

func (ctx *Context) elementsToArray(elements []*botc.MessageElement) []map[string]interface{} {
	var result []map[string]interface{}

	for _, elem := range elements {
		segment := make(map[string]interface{})
		data := make(map[string]interface{})

		switch elem.Type {
		case botc.TextElement:
			segment["type"] = "text"
			data["text"] = elem.Content
		case botc.ImageElement:
			segment["type"] = "image"
			// Get resource file path from resource ID
			if resource, err := ctx.service.grb.GetResource(elem.Source); err == nil {
				data["file"] = fmt.Sprintf("file://%s", resource.FilePath)
			} else {
				// Fallback to source as URL
				data["file"] = elem.Source
			}
		case botc.VoiceElement:
			segment["type"] = "record"
			// Get resource file path from resource ID
			if resource, err := ctx.service.grb.GetResource(elem.Source); err == nil {
				data["file"] = fmt.Sprintf("file://%s", resource.FilePath)
			} else {
				// Fallback to source as URL
				data["file"] = elem.Source
			}
		case botc.VideoElement:
			segment["type"] = "video"
			// Get resource file path from resource ID
			if resource, err := ctx.service.grb.GetResource(elem.Source); err == nil {
				data["file"] = fmt.Sprintf("file://%s", resource.FilePath)
			} else {
				// Fallback to source as URL
				data["file"] = elem.Source
			}
		case botc.MentionElement:
			segment["type"] = "at"
			data["qq"] = elem.Source
		case botc.StickerElement:
			segment["type"] = "face"
			data["id"] = elem.Source
		case botc.QuoteElement:
			segment["type"] = "reply"
			data["id"] = elem.Source
		default:
			continue
		}

		segment["data"] = data
		result = append(result, segment)
	}

	return result
}