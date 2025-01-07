package lagrange

import (
	"fmt"
	"github.com/Jel1ySpot/GoroBot/pkg/core"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
	lgrEntity "github.com/LagrangeDev/LagrangeGo/client/entity"
	"strconv"
	"strings"
)

type Context struct {
	service *Service
}

func (ctx *Context) ID() string {
	return strconv.FormatInt(int64(ctx.service.config.Account.Uin), 10)
}

func (ctx *Context) Name() string {
	return "Lagrange-adapter"
}

func (ctx *Context) Protocol() string {
	return "lagrange"
}

func (ctx *Context) Status() GoroBot.LoginStatus {
	return ctx.service.status
}

func (ctx *Context) SendDirectMessage(target entity.User, elements []*message.Element) error {
	uin, err := strconv.ParseUint(target.ID, 10, 32)
	if err != nil {
		return err
	}

	elems := FromBaseMessage(elements)

	if _, err := ctx.service.qqClient.SendPrivateMessage(uint32(uin), elems); err != nil {
		return err
	}
	return nil
}

func (ctx *Context) SendGroupMessage(target entity.Group, elements []*message.Element) error {
	uin, err := strconv.ParseUint(target.ID, 10, 32)
	if err != nil {
		return err
	}

	elems := FromBaseMessage(elements)

	if _, err := ctx.service.qqClient.SendGroupMessage(uint32(uin), elems); err != nil {
		return err
	}
	return nil
}

func (ctx *Context) GetMessageFileUrl(msg *message.Base) (string, error) {
	var elem *message.Element
	for _, elem = range msg.Elements {
		if elem.Type == message.File {
			break
		}
	}
	if elem == nil || elem.Type != message.File {
		return "", fmt.Errorf("file element not exist")
	}

	var (
		msgProtocol string
		fileDetail  string
	)
	if n, err := fmt.Sscanf(elem.Source, "%s:%s", &msgProtocol, &fileDetail); err != nil || n != 2 {
		return "", fmt.Errorf("invalid source format")
	}
	args := strings.Split(fileDetail, "&")
	if msgProtocol != "lagrange" {
		return "", fmt.Errorf("protocol not match")
	}
	switch msg.MessageType {
	case message.DirectMessage:
		return ctx.service.qqClient.GetPrivateFileURL(args[0], args[1])
	case message.GroupMessage:
		id, err := strconv.ParseUint(args[0], 10, 32)
		if err != nil {
			return "", fmt.Errorf("parse file detail error: %v", err)
		}
		return ctx.service.qqClient.GetGroupFileURL(uint32(id), args[1])
	}
	return "", nil
}

func (ctx *Context) Contacts() []entity.User {
	data, err := ctx.service.qqClient.GetFriendsData()
	if err != nil {
		return nil
	}

	var users []entity.User

	for uin, user := range data {
		users = append(users, entity.User{
			Base: &entity.Base{
				ID:     strconv.FormatUint(uint64(uin), 10),
				Name:   user.Nickname,
				Avatar: user.Avatar,
			},
			Nickname:  user.Remarks,
			Age:       user.Age,
			Authority: entity.Member,
		})
	}
	return users
}

func (ctx *Context) Groups() []entity.Group {
	data, err := ctx.service.qqClient.GetAllGroupsInfo()
	if err != nil {
		return nil
	}

	membersData, err := ctx.service.qqClient.GetAllGroupsMembersData()
	if err != nil {
		membersData = nil
	}

	var groups []entity.Group

	for uin, group := range data {
		groups = append(groups, entity.Group{
			Base: &entity.Base{
				ID:     strconv.FormatUint(uint64(uin), 10),
				Name:   group.GroupName,
				Avatar: group.Avatar(),
			},
			Members: func() []*entity.User {
				if membersData == nil {
					return nil
				}
				if _, ok := membersData[uin]; !ok {
					return nil
				}
				var members []*entity.User
				for uin, member := range membersData[uin] {
					members = append(members, &entity.User{
						Base: &entity.Base{
							ID:     strconv.FormatUint(uint64(uin), 10),
							Name:   member.Nickname,
							Avatar: member.Avatar,
						},
						Nickname: member.DisplayName(),
						Age:      0,
						Authority: func() entity.Authority {
							switch member.Permission {
							case lgrEntity.Member:
								return entity.Member
							case lgrEntity.Admin:
								return entity.GroupAdmin
							case lgrEntity.Owner:
								return entity.GroupOwner
							}
							return entity.Member
						}(),
					})
				}
				return members
			}(),
		})
	}
	return groups
}
