package command

import (
	"fmt"
	"strings"

	botc "github.com/Jel1ySpot/GoroBot/pkg/core/bot_context"
	"github.com/google/shlex"
)

type Context struct {
	botc.MessageContext
	argQueue  []string
	raw       string
	Commands  []string
	Arguments []string
	KvArgs    map[string]string
	Options   map[string]string
}

func NewCommandContext(msg botc.MessageContext, text string) *Context {
	tokens, err := shlex.Split(text)
	if err != nil {
		tokens = strings.Fields(text)
	}

	return &Context{
		MessageContext: msg,
		argQueue:       tokens,
		raw:            text,
		Arguments:      []string{},
		KvArgs:         make(map[string]string),
		Options:        make(map[string]string),
	}
}

func (ctx *Context) Clone() *Context {
	argQueue := append([]string(nil), ctx.argQueue...)
	commands := append([]string(nil), ctx.Commands...)
	arguments := append([]string(nil), ctx.Arguments...)

	kvArgs := make(map[string]string, len(ctx.KvArgs))
	for k, v := range ctx.KvArgs {
		kvArgs[k] = v
	}

	options := make(map[string]string, len(ctx.Options))
	for k, v := range ctx.Options {
		options[k] = v
	}

	return &Context{
		MessageContext: ctx.MessageContext,
		argQueue:       argQueue,
		raw:            ctx.raw,
		Commands:       commands,
		Arguments:      arguments,
		KvArgs:         kvArgs,
		Options:        options,
	}
}

func (ctx *Context) processTokens(schema *Schema) error {
	ctx.Commands = nil
	ctx.Arguments = nil
	for k := range ctx.Options {
		delete(ctx.Options, k)
	}

	queue := append([]string(nil), ctx.argQueue...)
	if len(queue) == 0 {
		return fmt.Errorf("unmatched command")
	}

	if !schema.Match(queue[0]) {
		return fmt.Errorf("unmatched command")
	}

	ctx.Commands = append(ctx.Commands, schema.Name)
	queue = queue[1:]

	currentSchema := schema
	argIndex := 0

	for len(queue) > 0 {
		token := queue[0]

		if s, ok := getSchemaFromSlice(&currentSchema.SubCommandSchemas, token); ok {
			ctx.Commands = append(ctx.Commands, s.Name)
			currentSchema = s
			queue = queue[1:]
			argIndex = 0
			continue
		}

		if token == "--" {
			ctx.Arguments = append(ctx.Arguments, queue[1:]...)
			queue = nil
			break
		}

		if strings.HasPrefix(token, "--") {
			queue = queue[1:]
			key, value, err := parseLongOption(token, &queue, currentSchema)
			if err != nil {
				return err
			}
			ctx.Options[key] = value
			continue
		}

		if strings.HasPrefix(token, "-") {
			queue = queue[1:]
			options, err := parseShortOption(token, &queue, currentSchema)
			if err != nil {
				return err
			}
			for k, v := range options {
				ctx.Options[k] = v
			}
			continue
		}

		ctx.Arguments = append(ctx.Arguments, token)
		if argIndex < len(currentSchema.Arguments) {
			arg := currentSchema.Arguments[argIndex]
			if !CheckInputType(token, arg.Type) {
				return fmt.Errorf("argument '%s' expected type '%s', received '%s'", arg.Name, arg.Type, token)
			}
			ctx.KvArgs[arg.Name] = token
		}
		argIndex++
		queue = queue[1:]
	}

	for i := argIndex; i < len(currentSchema.Arguments); i++ {
		if currentSchema.Arguments[i].Required {
			return fmt.Errorf("argument '%s' is required", currentSchema.Arguments[i].Name)
		}
	}

	for _, opt := range currentSchema.Options {
		if _, ok := ctx.Options[opt.Name]; ok {
			continue
		}
		if opt.Required {
			return fmt.Errorf("option %s is required", opt.Name)
		}
		if opt.Default != "" {
			ctx.Options[opt.Name] = opt.Default
		}
	}

	for k := range ctx.Options {
		if _, ok := currentSchema.getOption(k); !ok {
			return fmt.Errorf("option '%s' not found", k)
		}
	}

	return nil
}

func parseLongOption(token string, argQueue *[]string, schema *Schema) (string, string, error) {
	key := token
	value := ""

	if strings.Contains(token, "=") {
		kv := strings.SplitN(token, "=", 2)
		key = kv[0]
		value = kv[1]
	}

	opt, ok := schema.getOption(key)
	if !ok {
		return "", "", fmt.Errorf("option '%s' not found", key)
	}

	if opt.Type == Boolean {
		return opt.Name, "true", nil
	}

	if value == "" {
		if len(*argQueue) == 0 {
			return "", "", fmt.Errorf("option '%s' expected at least one argument", key)
		}
		value = (*argQueue)[0]
		*argQueue = (*argQueue)[1:]
	}

	if !CheckInputType(value, opt.Type) {
		return "", "", fmt.Errorf("option '%s' expected type '%s', received '%s'", key, opt.Type, value)
	}

	return opt.Name, value, nil
}

func parseShortOption(token string, argQueue *[]string, schema *Schema) (map[string]string, error) {
	options := make(map[string]string)

	chars := token[1:]

	for i, char := range chars {
		key := "-" + string(char)

		opt, ok := schema.getOption(key)
		if !ok {
			return nil, fmt.Errorf("option '%s' not found", key)
		}

		if opt.Type == Boolean {
			options[opt.Name] = "true"
			continue
		}

		if i != len(chars)-1 || len(*argQueue) == 0 {
			return nil, fmt.Errorf("option '%s' expected at least one argument", key)
		}

		value := (*argQueue)[0]
		if !CheckInputType(value, opt.Type) {
			return nil, fmt.Errorf("option '%s' expected type '%s', received '%s'", key, opt.Type, value)
		}

		options[opt.Name] = value
		*argQueue = (*argQueue)[1:]
	}

	return options, nil
}
