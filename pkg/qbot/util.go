package qbot

import (
	"fmt"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/tencent-connect/botgo/dto"
	"strings"
)

func FormatUserID(uin uint32) string {
	return fmt.Sprintf("qbot:user&%d", uin)
}

func FormatID(type_ string, v ...string) string {
	return fmt.Sprintf("qbot:%s&%s", type_, strings.Join(v, "&"))
}

func (s *Service) GenResourceURL(id string) string {
	return fmt.Sprintf("%s/resource/%s", s.config.Http.BaseURL, id)
}

func ParseUser(user *dto.User, member *dto.Member) *entity.Sender {
	if user == nil {
		return nil
	}
	sender := &entity.Sender{
		User: &entity.User{
			Base: &entity.Base{
				ID:     FormatID("User", user.ID),
				Name:   user.Username,
				Avatar: user.Avatar,
			},
		},
	}
	if member != nil {
		sender.Nickname = member.Nick
	} else {
		sender.Nickname = sender.Name
	}
	return sender
}
