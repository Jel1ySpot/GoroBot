package message

import (
	"encoding/json"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"time"
)

type Type int

const (
	DirectMessage Type = iota
	GroupMessage
)

type Base struct {
	MessageType Type
	ID          string
	Content     string
	Elements    []*Element
	Sender      entity.Sender
	Time        time.Time
}

func (m *Base) Marshall() string {
	bytes, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

func UnmarshallMessage(str string) (*Base, error) {
	msg := Base{}
	if err := json.Unmarshal([]byte(str), &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
