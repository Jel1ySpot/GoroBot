package lagrange

import (
	"fmt"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	LgrMessage "github.com/LagrangeDev/LagrangeGo/message"
	"strconv"
	"time"
	"unsafe"
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

func ReplyElementToMessage(service *Service, elem *LgrMessage.ReplyElement) *botc.BaseMessage {
	var event any

	isGroup := elem.GroupUin > 0
	if isGroup {
		event = &LgrMessage.GroupMessage{
			ID:       elem.ReplySeq,
			GroupUin: elem.GroupUin,
			Sender: &LgrMessage.Sender{
				Uin: elem.SenderUin,
			},
			Time:     elem.Time,
			Elements: elem.Elements,
		}
	} else {
		event = &LgrMessage.PrivateMessage{
			ID: elem.ReplySeq,
			Sender: &LgrMessage.Sender{
				Uin: elem.SenderUin,
				UID: elem.SenderUID,
			},
			Time:     elem.Time,
			Elements: elem.Elements,
		}
	}
	return &botc.BaseMessage{
		MessageType: botc.MessageType(int(*(*byte)(unsafe.Pointer(&isGroup)))),
		ID:          GenMsgSeqID(elem.ReplySeq),
		Content:     LgrMessage.ToReadableString(elem.Elements),
		Elements:    ParseElementsFromEvent(service, event),
		Sender: &entity.Sender{
			User: &entity.User{
				Base: &entity.Base{
					ID: GenUserID(elem.SenderUin),
				},
			},
			From: &entity.Base{
				ID: GenGroupID(elem.GroupUin),
			},
		},
		Time: time.Unix(int64(elem.Time), 0),
	}
}

func SenderConv(u *LgrMessage.Sender, group *LgrMessage.GroupMessage) *entity.Sender {
	var from *entity.Base
	if group != nil {
		from = &entity.Base{
			ID:     GenGroupID(group.GroupUin),
			Name:   group.GroupName,
			Avatar: "",
		}
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
