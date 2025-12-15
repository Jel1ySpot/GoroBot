package command

import (
	"regexp"
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

func NewCommandFormatBuilder(name string, system *System) *FormatBuilder {
	return &FormatBuilder{
		system:   system,
		registry: &Registry{Schema: Schema{Name: name}},
		err:      nil,
	}
}

func (f *FormatBuilder) SubCommand(name string) *FormatBuilder {
	f.registry.SubRegistries = append(f.registry.SubRegistries, Registry{
		Schema: Schema{Name: name},
	})
	child := &FormatBuilder{
		system:   f.system,
		registry: &f.registry.SubRegistries[len(f.registry.SubRegistries)-1],
		parent:   f,
		err:      f.err,
	}

	return child
}

func (f *FormatBuilder) Option(name string, shortName string, inputType InputType, required bool, Default string, help string) *FormatBuilder {
	if f.err != nil {
		return f
	}
	f.registry.Schema.Options = append(f.registry.Schema.Options, SchemaOption{
		Name:      name,
		ShortName: shortName,
		Type:      inputType,
		Help:      help,
		Required:  required,
		Default:   Default,
	})
	return f
}

func (f *FormatBuilder) Argument(name string, inputType InputType, required bool, help string) *FormatBuilder {
	if f.err != nil {
		return f
	}
	f.registry.Schema.Arguments = append(f.registry.Schema.Arguments, SchemaArgument{
		Name:     name,
		Type:     inputType,
		Help:     help,
		Required: required,
	})
	return f
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
