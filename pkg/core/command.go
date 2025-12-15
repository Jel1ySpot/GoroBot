package GoroBot

import "github.com/Jel1ySpot/GoroBot/pkg/core/command"

func (i *Instant) Command(name string) *command.FormatBuilder {
	return command.NewCommandFormatBuilder(name, i.commands)
}
