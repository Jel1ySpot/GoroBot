package GoroBot

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
)

type BotContext interface {
	ID() string
	Name() string
	Protocol() string
	Status() LoginStatus
	Contacts() []entity.User
	Groups() []entity.Group
	SendDirectMessage(target entity.User, message []*message.Element) error
	SendGroupMessage(target entity.Group, message []*message.Element) error
	GetMessageFileUrl(msg *message.Base) (string, error)
}

type LoginStatus int

const (
	Offline LoginStatus = iota
	Online
	Connect
	Disconnect
	Reconnect
)
