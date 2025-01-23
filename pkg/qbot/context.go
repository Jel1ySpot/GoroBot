package qbot

import (
	"context"
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/Jel1ySpot/GoroBot/pkg/core/entity"
)

type Context struct {
	*Service
}

func NewContext(service *Service) *Context {
	return &Context{service}
}

func (c *Context) ID() string {
	u, err := c.api.Me(context.Background())
	if err != nil {
		return c.config.Credentials.AppID
	}
	return u.ID
}

func (c *Context) Name() string {
	return "qbot"
}

func (c *Context) Protocol() string {
	return "qbot"
}

func (c *Context) Status() botc.LoginStatus {
	return c.status
}

func (c *Context) NewMessageBuilder() botc.MessageBuilder {
	return NewMessageBuilder(&MessageContext{
		bot:     c,
		message: nil,
	}, c.Service)
}

func (c *Context) SendDirectMessage(target entity.User, elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	//TODO implement me
	return nil, nil
}

func (c *Context) SendGroupMessage(target entity.Group, elements []*botc.MessageElement) (*botc.BaseMessage, error) {
	//TODO implement me
	return nil, nil
}

func (c *Context) Contacts() []entity.User {
	return nil
}

func (c *Context) Groups() []entity.Group {
	return nil
}

func (c *Context) GetMessageFileUrl(msg *botc.BaseMessage) (string, error) {
	return "", nil
}
