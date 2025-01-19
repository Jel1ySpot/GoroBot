package bot_context

import (
	"encoding/json"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"time"
)

type MessageType int

const (
	DirectMessage MessageType = iota
	GroupMessage
)

type BaseMessage struct {
	MessageType MessageType
	ID          string
	Content     string
	Elements    []*MessageElement
	Sender      *entity.Sender
	Time        time.Time
}

func (m *BaseMessage) Marshall() string {
	bytes, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

func UnmarshallMessage(str string) (*BaseMessage, error) {
	msg := BaseMessage{}
	if err := json.Unmarshal([]byte(str), &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
