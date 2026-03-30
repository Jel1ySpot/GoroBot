package GoroBot

import "github.com/Jel1ySpot/GoroBot/pkg/core/command"

func (i *Instant) Command(name string) *command.FormatBuilder {
	return command.NewCommandFormatBuilder(name, i.commands)
}

// GetCommandSchemas 返回所有已注册的顶级命令 Schema
func (i *Instant) GetCommandSchemas() []command.Schema {
	return i.commands.GetSchemas()
}
