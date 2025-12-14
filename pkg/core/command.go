package GoroBot

import "github.com/Jel1ySpot/GoroBot/pkg/core/command"

func (i *Instant) Command(format string) *command.FormatBuilder {
	return command.NewCommandFormatBuilder(format, i.commands)
}
