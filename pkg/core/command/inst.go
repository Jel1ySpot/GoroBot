package command

import (
	"fmt"
	"strings"
)

type Inst struct {
	Name      string
	Options   []Option
	Arguments []Argument
	Alias     map[string]map[string]string
	Subs      map[string]Inst
	handler   func(ctx *Context)
}

type Option struct {
	Type    ArgType
	Name    string
	Short   string
	Default string
}

type Argument struct {
	Type     ArgType
	Required bool
	Name     string
	Default  string
}

type Alias map[string]map[string]string

type ArgType int

const (
	StringArg ArgType = iota
	TextArg
)

func ParseArgType(s string) (ArgType, error) {
	if s == "" {
		return StringArg, nil
	}
	switch strings.ToLower(s) {
	case "string":
		return StringArg, nil
	case "text":
		return TextArg, nil
	default:
		return StringArg, fmt.Errorf("unknown arg type: %s", s)
	}
}

func ParseCommand(test string) (name, last string) {
	test = strings.TrimSpace(test)
	split := strings.Split(test, " ")
	if len(split) == 0 {
		return "", ""
	}
	name = split[0]
	last = strings.TrimSpace(strings.TrimLeft(name, test))
	return
}
