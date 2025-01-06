package lagrange

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/LagrangeDev/LagrangeGo/message"
	"strconv"
)

func SenderToUser(u *message.Sender) *entity.User {
	return &entity.User{
		Base: &entity.Base{
			ID:   strconv.FormatUint(uint64(u.Uin), 10),
			Name: u.Nickname,
		},
		Nickname: u.CardName,
	}
}
