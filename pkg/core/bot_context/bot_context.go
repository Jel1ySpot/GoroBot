package bot_context

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
)

type BotContext interface {
	ID() string
	Name() string
	Protocol() string
	Status() LoginStatus
	NewMessageBuilder() MessageBuilder
	SendDirectMessage(target entity.User, elements []*MessageElement) (*BaseMessage, error)
	SendGroupMessage(target entity.Group, elements []*MessageElement) (*BaseMessage, error)
	Contacts() []entity.User
	Groups() []entity.Group
	GetMessageFileUrl(msg *BaseMessage) (string, error)
}

type LoginStatus int

const (
	Offline LoginStatus = iota
	Online
	Connect
	Disconnect
	Reconnect
)
