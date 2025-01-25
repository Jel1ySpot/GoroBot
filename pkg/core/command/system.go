package command

import (
	"bufio"
	"fmt"
	"github.com/google/shlex"
	"github.com/google/uuid"
	"io"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"unsafe"
)

type System struct {
	Commands map[string]*Registry
	mu       sync.Mutex
}

type Registry struct {
	id  string
	cmd Inst
}

func NewCommandRegistry(cmd Inst) *Registry {
	id := uuid.New().String()
	return &Registry{
		id:  id,
		cmd: cmd,
	}
}

func NewCommandSystem() *System {
	return &System{
		Commands: make(map[string]*Registry),
		mu:       sync.Mutex{},
	}
}

func (s *System) Register(cmd Inst) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	registry := NewCommandRegistry(cmd)
	s.Commands[registry.id] = registry
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.Commands, registry.id)
	}
}

func (s *System) Emit(cmdCtx *Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, registry := range s.Commands {
		registry.Emit(cmdCtx)
	}
}

func (r *Registry) Emit(cmdCtx *Context) { // 触发指令Reg
	if err := r.cmd.Emit(cmdCtx); err != nil {
		if err.Error() != "unmatched command" {
			_, _ = cmdCtx.ReplyText(err.Error())
		}
	}
}

func (i Inst) Emit(cmdCtx *Context) error {
	err := i.parse(cmdCtx)
	if cmdCtx.Command != i.Name {
		return err
	}
	for _, i := range i.Subs {
		if err := i.Emit(cmdCtx); err == nil {
			return nil
		}
	}

	if err != nil {
		return err
	}

	if i.handler != nil {
		go i.handler(cmdCtx)
	}

	return nil
}

func (r *Registry) CheckAlias(cmdCtx *Context) {
	_ = r.cmd.CheckAlias(cmdCtx)
}

func (i Inst) CheckAlias(cmdCtx *Context) error { // 在消息事件触发时调用
	for _, i := range i.Subs {
		if i.CheckAlias(cmdCtx) == nil {
			return nil
		}
	}
	for alias, opts := range i.Alias {
		reg := regexp.MustCompile(alias)
		if !reg.MatchString(cmdCtx.ArgumentString) {
			return fmt.Errorf("alias %s does not match %s", alias, cmdCtx.ArgumentString)
		}
		// 如果匹配指令别名
		cmdCtx.Command = i.Name
		matches := reg.FindStringSubmatch(cmdCtx.ArgumentString) // 正则中的子串
		if opts == nil {
			if i.handler != nil {
				i.handler(cmdCtx)
			}
			continue
		}
		for _, arg := range i.Arguments { // 遍历参数
			if val, ok := opts[arg.Name]; ok {
				if strings.HasPrefix(val, "$") { // 如果格式为 "$SubExpName"
					if i := reg.SubexpIndex(val[1:]); i != -1 { // 如果子串存在
						cmdCtx.Options[arg.Name] = matches[i]
						continue
					}
				}
				cmdCtx.Options[arg.Name] = val
			} else { // 如果不存在值，则默认
				cmdCtx.Options[arg.Name] = arg.Default
			}
		}
		for _, opt := range i.Options { // 遍历选项
			if val, ok := opts[opt.Name]; ok { // 如果别名选项中存在值
				if strings.HasPrefix(val, "$") { // 如果格式为 "$SubExpName"
					if i := reg.SubexpIndex(val[1:]); i != -1 { // 如果子串存在
						cmdCtx.Options[opt.Name] = matches[i]
						continue
					}
				}
				cmdCtx.Options[opt.Name] = val
			} else { // 如果不存在值，则默认
				cmdCtx.Options[opt.Name] = opt.Default
			}
		}
		if i.handler != nil {
			i.handler(cmdCtx)
		}
	}
	return nil
}

func (i Inst) parse(cmd *Context) error {
	name, last := ParseCommand(cmd.ArgumentString)
	if name != i.Name {
		return fmt.Errorf("unmatched command")
	}
	cmd.Command = name
	cmd.ArgumentString = last
	var reader io.Reader = strings.NewReader(cmd.ArgumentString)
	l := shlex.NewLexer(reader)
	reader = getReader(l)

	var (
		now string
		err error
	)

	argIndex := 0
	for {
		if argIndex < len(i.Arguments) && i.Arguments[argIndex].Type == TextArg {
			text, err := io.ReadAll(reader)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("error occurred while parsing argument '%s'", i.Arguments[argIndex].Name)
			}
			if strings.HasPrefix(string(text), "-") {
				l = shlex.NewLexer(strings.NewReader(string(text)))
				reader = getReader(l)
			} else {
				cmd.Args[i.Arguments[argIndex].Name] = string(text)
				return nil
			}
		}

		now, err = l.Next()
		if err != nil {
			break
		}
		if strings.HasPrefix(now, "-") {
			optName := strings.TrimLeft(now, "-")
			isShort := len(now)-len(optName) == 1
			for _, opt := range i.Options {
				if (isShort && opt.Short == optName) || opt.Name == optName {
					if opt.Type == TextArg {
						text, err := io.ReadAll(reader)
						if err != nil {
							return fmt.Errorf("error occurred while parsing option '%s'", optName)
						}
						cmd.Options[opt.Name] = string(text)
						return nil
					}
					now, err = l.Next()
					if err != nil {
						return fmt.Errorf("error occurred while parsing option '%s'", optName)
					}
					cmd.Options[opt.Name] = now
					optName = ""
					break
				}
			}
			if optName != "" {
				return fmt.Errorf("unknown option '%s'", strings.TrimLeft(now, "-"))
			}
			continue
		}
		if argIndex < len(i.Arguments) {
			cmd.Args[i.Arguments[argIndex].Name] = now
			argIndex++
		} else {
			return fmt.Errorf("too many arguments")
		}
	}

	if err == io.EOF {
		return i.checkRequired(cmd)
	} else {
		return fmt.Errorf("error occurred while parsing command '%v'", err)
	}
}

func (i Inst) checkRequired(cmdCtx *Context) error {
	for _, arg := range i.Arguments {
		if _, ok := cmdCtx.Args[arg.Name]; !ok {
			if arg.Required {
				return fmt.Errorf("missing required argument '%s'", arg.Name)
			} else {
				cmdCtx.Args[arg.Name] = arg.Default
			}
		}
	}

	for _, opt := range i.Options {
		if _, ok := cmdCtx.Options[opt.Name]; !ok {
			cmdCtx.Options[opt.Name] = opt.Default
		}
	}
	return nil
}

func getReader(lexer *shlex.Lexer) *bufio.Reader {
	v := reflect.ValueOf(lexer).Elem()

	field := v.FieldByName("input")

	if field.IsValid() {
		// 读取未导出字段的值
		return (*bufio.Reader)(unsafe.Pointer(field.UnsafeAddr()))
	}
	return nil
}
