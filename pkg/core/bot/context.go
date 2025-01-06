package bot

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
)

type Context interface {
	ID() string
	Name() string
	Protocol() string
	Status() LoginStatus
	SendDirectMessage(target entity.User, message []*message.Element) error
	SendGroupMessage(target entity.Group, message []*message.Element) error
	Contacts() []entity.User
	Groups() []entity.Group
}

type LoginStatus int

const (
	Offline LoginStatus = iota
	Online
	Connect
	Disconnect
	Reconnect
)
