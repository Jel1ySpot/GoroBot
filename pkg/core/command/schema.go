package command

import "strings"

type InputType string

type Schema struct {
	Name              string
	Options           []SchemaOption
	Arguments         []SchemaArgument
	SubCommandSchemas []Schema
}

type SchemaOption struct {
	Name      string
	ShortName string
	Type      InputType
	Help      string
	Default   string
	Required  bool
}

type SchemaArgument struct {
	Name string
	Type InputType
	Help string
}

func NewSchema(name string) *Schema {
	return &Schema{
		Name: name,
	}
}

func (s *Schema) AddOption(short string, name string, inputType InputType, help string) *Schema {
	s.Options = append(s.Options, SchemaOption{
		Name:      name,
		ShortName: short,
		Type:      inputType,
		Help:      help,
	})
	return s
}

func (s *Schema) AddArgument(name string, inputType InputType, help string) *Schema {
	s.Arguments = append(s.Arguments, SchemaArgument{
		Name: name,
		Type: inputType,
		Help: help,
	})
	return s
}

func (s *Schema) AddSubCommandSchema(schema Schema) *Schema {
	s.SubCommandSchemas = append(s.SubCommandSchemas, schema)
	return s
}

func (s *Schema) getOption(key string) (*SchemaOption, bool) {
	for _, option := range s.Options {
		if strings.ToLower(key) == strings.ToLower(option.Name) || strings.ToLower(key) == strings.ToLower(option.ShortName) {
			return &option, true
		}
	}
	return nil, false
}

func (s *Schema) Match(token string) bool {
	return strings.ToLower(token) == strings.ToLower(s.Name)
}
