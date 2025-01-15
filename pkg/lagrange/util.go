package lagrange

import (
	"fmt"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	LgrMessage "github.com/LagrangeDev/LagrangeGo/message"
	"strconv"
)

func CheckMessageType(msg any) botc.MessageType {
	switch msg.(type) {
	case *LgrMessage.PrivateMessage:
		return botc.DirectMessage
	case *LgrMessage.GroupMessage:
		return botc.GroupMessage
	}
	return -1
}

func SenderConv(u *LgrMessage.Sender, group *LgrMessage.GroupMessage) *entity.Sender {
	from := ""
	if group != nil {
		from = GenGroupID(group.GroupUin)
	}
	return &entity.Sender{
		User: &entity.User{
			Base: &entity.Base{
				ID:   GenUserID(u.Uin),
				Name: u.Nickname,
			},
			Nickname: u.CardName,
		},
		From: from,
	}
}

func GenUserID(uin uint32) string {
	return fmt.Sprintf("lagrange:user&%d", uin)
}

func ParseUin(str string) (uint32, bool) {
	info, ok := entity.ParseInfo(str)
	if !ok || info.Protocol != "lagrange" {
		return 0, false
	}
	id, err := strconv.ParseUint(info.Args[1], 10, 32)
	if err != nil {
		return 0, false
	}
	return uint32(id), true
}

func GenGroupID(uin uint32) string {
	return fmt.Sprintf("lagrange:group&%d", uin)
}

func GenMsgSeqID(uin uint32) string {
	return fmt.Sprintf("lagrange:msg_seq&%d", uin)
}
