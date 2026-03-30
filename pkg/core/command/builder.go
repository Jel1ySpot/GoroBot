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
	release  func()
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

func (f *FormatBuilder) Action(handler Handler) *FormatBuilder {
	f.registry.Handler = handler
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
	root := f
	for root.parent != nil {
		root = root.parent
	}

	if root.err != nil {
		return func() {}, root.err
	}

	// 如果已经注册过，先移除旧的注册，确保最新结构生效
	if root.release != nil {
		root.release()
	}

	reg := *root.registry
	reg.Schema = syncSchema(root.registry)
	root.release = root.system.Register(reg)
	return root.release, nil
}

func syncSchema(reg *Registry) Schema {
	schema := reg.Schema
	schema.SubCommandSchemas = nil
	for i := range reg.SubRegistries {
		childSchema := syncSchema(&reg.SubRegistries[i])
		schema.SubCommandSchemas = append(schema.SubCommandSchemas, childSchema)
	}
	return schema
}
