package lagrange

import (
	"fmt"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
	lgrEntity "github.com/LagrangeDev/LagrangeGo/client/entity"
	"strconv"
)

type Context struct {
	service *Service
}

func (ctx *Context) ID() string {
	return fmt.Sprintf("%d", ctx.service.config.Account.Uin)
}

func (ctx *Context) Name() string {
	return "Lagrange"
}

func (ctx *Context) Protocol() string {
	return "lagrange"
}

func (ctx *Context) Status() botc.LoginStatus {
	return ctx.service.status
}

func (ctx *Context) NewMessageBuilder() botc.MessageBuilder {
	return &MessageBuilder{
		service: ctx.service,
	}
}

func (ctx *Context) SendDirectMessage(target entity.User, elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	uin, ok := ParseUin(target.ID)
	if !ok {
		return nil, fmt.Errorf("invalid uin %s", target.ID)
	}

	elems := TranslateMessageElement(ctx.service, elements)

	if msg, err := ctx.service.qqClient.SendPrivateMessage(uint32(uin), elems); err != nil {
		return nil, err
	} else {
		return ParseMessageEvent(ctx.service, msg)
	}
}

func (ctx *Context) SendGroupMessage(target entity.Group, elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	uin, ok := ParseUin(target.ID)
	if !ok {
		return nil, fmt.Errorf("invalid uin %s", target.ID)
	}

	elems := TranslateMessageElement(ctx.service, elements)

	if msg, err := ctx.service.qqClient.SendGroupMessage(uint32(uin), elems); err != nil {
		return nil, err
	} else {
		return ParseMessageEvent(ctx.service, msg)
	}
}

func (ctx *Context) GetMessageFileUrl(msg *botc.BaseMessage) (string, error) {
	var elem *botc.MessageElement
	for _, elem = range msg.Elements {
		if elem.Type == botc.FileElement {
			break
		}
	}
	if elem == nil || elem.Type != botc.FileElement {
		return "", fmt.Errorf("file element not exist")
	}

	info, ok := entity.ParseInfo(elem.Source)
	if !ok || info.Protocol != "lagrange" {
		return "", fmt.Errorf("invalid source")
	}
	switch msg.MessageType {
	case botc.DirectMessage:
		return ctx.service.qqClient.GetPrivateFileURL(info.Args[0], info.Args[1])
	case botc.GroupMessage:
		id, err := strconv.ParseUint(info.Args[0], 10, 32)
		if err != nil {
			return "", fmt.Errorf("parse file detail error: %v", err)
		}
		return ctx.service.qqClient.GetGroupFileURL(uint32(id), info.Args[1])
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
				ID:     GenUserID(uin),
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
				ID:     GenGroupID(uin),
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
							ID:     GenUserID(uin),
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
