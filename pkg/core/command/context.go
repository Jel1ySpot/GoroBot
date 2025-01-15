package command

import (
	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
)

type Context struct {
	botc.MessageContext
	Tokens         []string
	Command        string
	ArgumentString string
	Args           map[string]string
	Options        map[string]string
}

func NewCommandContext(msg botc.MessageContext, text string) *Context {
	return &Context{
		MessageContext: msg,
		ArgumentString: text,
		Args:           make(map[string]string),
		Options:        make(map[string]string),
	}
}
