package command

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
)

type Context struct {
	message.Context
	Tokens         []string
	Command        string
	ArgumentString string
	Args           map[string]string
	Options        map[string]string
}

func NewCommandContext(msg message.Context, text string) *Context {
	return &Context{
		Context:        msg,
		ArgumentString: text,
		Args:           make(map[string]string),
		Options:        make(map[string]string),
	}
}
