package command

import (
	"fmt"
	"regexp"
)

var (
	RegFormat         = regexp.MustCompile(`^(?P<name>\w+?)(?P<args>(?: [<\[]\w+?(?::\w+?)?[>\]](?:=\w+?)? ?)*)$`)
	ArgumentRegFormat = regexp.MustCompile(`(?:<(?P<required_name>\w+?)(?::(?P<required_type>\w+?))?>|\[(?P<optional_name>\w+?)(?::(?P<optional_type>\w+?))?\])(?:=(?P<default>\w+))?`)
	OptionRegFormat   = regexp.MustCompile(`^-(?P<short>\w+?) \[(?P<name>\w+?):(?P<type>\w+?)\](?:=(?P<default>\w+?))?$`)
)

type FormatBuilder struct {
	format   Format
	callback func(Format) func()
	err      error
}

func NewCommandFormatBuilder(format string, callback func(Format) func()) *FormatBuilder {
	matches := RegFormat.FindStringSubmatch(format)

	if matches == nil {
		return &FormatBuilder{
			err: fmt.Errorf("invalid format \"%s\"", format),
		}
	}

	var (
		nameIndex = RegFormat.SubexpIndex("name")
		argsIndex = RegFormat.SubexpIndex("args")
	)

	builder := FormatBuilder{
		format: Format{
			Name:  matches[nameIndex],
			Alias: make(map[string]map[string]string),
		},
		callback: callback,
	}
	return builder.Arguments(matches[argsIndex])
}

func (b *FormatBuilder) Build() (func(), error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.callback(b.format), nil
}

func (b *FormatBuilder) Arguments(args string) *FormatBuilder {
	arg := ArgumentRegFormat.FindString(args)
	if arg == "" {
		return b
	}

	matches := ArgumentRegFormat.FindStringSubmatch(arg)

	var (
		requiredNameIndex = ArgumentRegFormat.SubexpIndex("required_name")
		requiredTypeIndex = ArgumentRegFormat.SubexpIndex("required_type")
		optionalNameIndex = ArgumentRegFormat.SubexpIndex("optional_name")
		optionalTypeIndex = ArgumentRegFormat.SubexpIndex("optional_type")
		defaultIndex      = ArgumentRegFormat.SubexpIndex("default")
	)

	switch arg[0] {
	case '<':
		argType, err := ParseArgType(matches[requiredTypeIndex])
		if err != nil {
			b.err = fmt.Errorf("invalid argument type \"%s\"", matches[requiredTypeIndex])
			return b
		}
		b.format.Arguments = append(b.format.Arguments, Argument{
			Type:     argType,
			Required: true,
			Name:     matches[requiredNameIndex],
			Default:  matches[defaultIndex],
		})
	case '[':
		argType, err := ParseArgType(matches[optionalTypeIndex])
		if err != nil {
			b.err = fmt.Errorf("invalid argument type \"%s\"", matches[optionalTypeIndex])
			return b
		}
		b.format.Arguments = append(b.format.Arguments, Argument{
			Type:     argType,
			Required: false,
			Name:     matches[optionalNameIndex],
			Default:  matches[defaultIndex],
		})
	}

	return b.Arguments(args[len(arg):])
}

func (b *FormatBuilder) Option(opt string) *FormatBuilder {
	if b.err != nil {
		return b
	}

	matches := OptionRegFormat.FindStringSubmatch(opt)
	if matches == nil {
		b.err = fmt.Errorf("invalid option \"%s\"", opt)
		return b
	}

	var (
		shortIndex   = OptionRegFormat.SubexpIndex("short")
		nameIndex    = OptionRegFormat.SubexpIndex("name")
		typeIndex    = OptionRegFormat.SubexpIndex("type")
		defaultIndex = OptionRegFormat.SubexpIndex("default")
	)

	argType, err := ParseArgType(matches[typeIndex])
	if err != nil {
		b.err = fmt.Errorf("invalid option type \"%s\"", matches[typeIndex])
		return b
	}

	b.format.Options = append(b.format.Options, Option{
		Type:    argType,
		Name:    matches[nameIndex],
		Short:   matches[shortIndex],
		Default: matches[defaultIndex],
	})

	return b
}

func (b *FormatBuilder) Alias(reg string, option map[string]string) *FormatBuilder {
	b.format.Alias[reg] = option
	return b
}
