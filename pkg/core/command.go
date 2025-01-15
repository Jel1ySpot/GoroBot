package GoroBot

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/command"
)

func (i *Instant) Command(format string, handler func(cmd *command.Context)) *command.FormatBuilder {
	return command.NewCommandFormatBuilder(format, func(cmd command.Format) func() {
		return i.commands.Register(cmd, func(args ...interface{}) {
			handler(args[0].(*command.Context))
		})
	})
}
