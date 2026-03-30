package GoroBot

import "github.com/Jel1ySpot/GoroBot/pkg/core/command"

func (i *Instant) Command(name string) *command.FormatBuilder {
	return command.NewCommandFormatBuilder(name, i.commands)
}

// GetCommandSchemas 返回所有已注册的顶级命令 Schema
func (i *Instant) GetCommandSchemas() []command.Schema {
	i.commands.Mu.Lock()
	defer i.commands.Mu.Unlock()
	schemas := make([]command.Schema, 0, len(i.commands.Commands))
	for _, reg := range i.commands.Commands {
		schemas = append(schemas, reg.Schema)
	}
	return schemas
}
