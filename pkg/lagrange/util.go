package lagrange

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
	LgrMessage "github.com/LagrangeDev/LagrangeGo/message"
	"strconv"
)

func CheckMessageType(msg any) message.Type {
	switch msg.(type) {
	case *LgrMessage.PrivateMessage:
		return message.DirectMessage
	case *LgrMessage.GroupMessage:
		return message.GroupMessage
	}
	return -1
}

func SenderConv(u *LgrMessage.Sender, group *LgrMessage.GroupMessage) *entity.Sender {
	from := ""
	if group != nil {
		from = strconv.FormatUint(uint64(group.GroupUin), 10)
	}
	return &entity.Sender{
		User: &entity.User{
			Base: &entity.Base{
				ID:   strconv.FormatUint(uint64(u.Uin), 10),
				Name: u.Nickname,
			},
			Nickname: u.CardName,
		},
		From: from,
	}
}
