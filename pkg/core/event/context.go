package event

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"time"
)

type Context struct {
	Self   *entity.User
	Sender *entity.Sender
	Group  *entity.Group
	Time   time.Time
	Type   Type
	Event  any

	// 如果是消息事件这里是消息预览，其他事件则为事件名称
	Content string
}

type Type int

const (
	MessageEvent Type = iota
	NotificationEvent
	RequestEvent
)
