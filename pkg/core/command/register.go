package command

type Registry struct {
	Name          string
	Schema        Schema
	Handler       Handler
	Aliases       []alias
	SubRegistries []Registry
}

type ParsedArguments = []string

type ParsedOptions = map[string]string

type Handler = func(ctx *Context) error

func getSchemaFromSlice(commands *[]Schema, token string) (*Schema, bool) {
	for i := range *commands {
		cmd := &(*commands)[i]
		if cmd.Match(token) {
			return cmd, true
		}
	}
	return nil, false
}
