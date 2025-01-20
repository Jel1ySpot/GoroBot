package qbot

import (
	"fmt"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/tencent-connect/botgo/dto"
	"strings"
)

func ParseID(idInfo string) (string, bool) {
	info, ok := entity.ParseInfo(idInfo)
	if !ok || info.Protocol != "qbot" || len(info.Args) < 2 {
		return "", false
	}
	return info.Args[1], true
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
