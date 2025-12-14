package command

import (
	"fmt"
	"regexp"
	"strings"
)

type alias struct {
	pattern   *regexp.Regexp
	transform func(*Context) *Context
}

type FormatBuilder struct {
	system   *System
	registry *Registry
	parent   *FormatBuilder
	err      error
}

func NewCommandFormatBuilder(format string, system *System) *FormatBuilder {
	schema, err := parseFormat(format)
	return &FormatBuilder{
		system: system,
		registry: &Registry{
			Schema: *schema,
		},
		err: err,
	}
}

func (f *FormatBuilder) SubCommand(format string) *FormatBuilder {
	schema, err := parseFormat(format)
	f.registry.SubRegistries = append(f.registry.SubRegistries, Registry{
		Schema: *schema,
	})
	child := &FormatBuilder{
		system:   f.system,
		registry: &f.registry.SubRegistries[len(f.registry.SubRegistries)-1],
		parent:   f,
		err:      err,
	}

	if f.err != nil && child.err == nil {
		child.err = f.err
	}

	return child
}

func (f *FormatBuilder) Action(handler func(ctx *Context)) *FormatBuilder {
	f.registry.Handler = func(ctx *Context) error {
		handler(ctx)
		return nil
	}
	return f
}

func (f *FormatBuilder) Alias(pattern string, transform func(*Context) *Context) *FormatBuilder {
	if f.err != nil {
		return f
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		f.err = err
		return f
	}

	f.registry.Aliases = append(f.registry.Aliases, alias{
		pattern:   re,
		transform: transform,
	})

	return f
}

func (f *FormatBuilder) Build() (func(), error) {
	if f.err != nil {
		return func() {}, f.err
	}

	if f.parent != nil {
		return func() {}, nil
	}

	return f.system.Register(*f.registry), nil
}

func parseFormat(format string) (*Schema, error) {
	parts := strings.Fields(format)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid command format")
	}

	schema := Schema{
		Name: parts[0],
	}

	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "<") && strings.HasSuffix(part, ">") {
			content := strings.TrimSuffix(strings.TrimPrefix(part, "<"), ">")
			name := content
			inputType := String

			if strings.Contains(content, ":") {
				s := strings.SplitN(content, ":", 2)
				name = s[0]
				inputType = normalizeInputType(InputType(s[1]))
			}

			schema.AddArgument(name, inputType, "")
		}
	}

	return &schema, nil
}

func normalizeInputType(t InputType) InputType {
	switch strings.ToLower(string(t)) {
	case "", "string", "text":
		return String
	case "bool", "boolean":
		return Boolean
	case "number", "int", "integer", "float", "double":
		return Number
	default:
		return t
	}
}
