package command

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
)

type Context struct {
	message.Context
	Tokens []string
}
