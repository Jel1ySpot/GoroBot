package GoroBot

import "github.com/Jel1ySpot/GoroBot/pkg/core/command"

func (i *Instant) Command(format string, handler func(ctx BotContext, cmd *command.Context)) *command.FormatBuilder {
	return command.NewCommandFormatBuilder(format, func(cmd command.Format) func() {
		return i.commands.Register(cmd, func(args ...interface{}) {
			handler(args[0].(BotContext), args[1].(*command.Context))
		})
	})
}
