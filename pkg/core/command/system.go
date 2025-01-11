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
	id      string
	format  Format
	handler func(args ...interface{})
}

func NewCommandRegistry(format Format, handler func(args ...interface{})) *Registry {
	id := uuid.New().String()
	return &Registry{
		id:      id,
		format:  format,
		handler: handler,
	}
}

func NewCommandSystem() *System {
	return &System{
		Commands: make(map[string]*Registry),
		mu:       sync.Mutex{},
	}
}

func (s *System) Register(format Format, handler func(args ...interface{})) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	registry := NewCommandRegistry(format, handler)
	s.Commands[registry.id] = registry
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.Commands, registry.id)
	}
}

func (s *System) Emit(botCtx interface{}, cmdCtx *Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, registry := range s.Commands {
		registry.Emit(botCtx, cmdCtx)
	}
}

func (r *Registry) Emit(botCtx interface{}, cmdCtx *Context) {
	if err := r.parse(cmdCtx); err != nil {
		if cmdCtx.Command == r.format.Name {
			_ = cmdCtx.ReplyText(err.Error())
		}
		return
	}
	if err := r.checkRequired(cmdCtx); err != nil {
		_ = cmdCtx.ReplyText(err.Error())
		return
	}
	r.handler(botCtx, cmdCtx)
}

func (r *Registry) CheckAlias(botCtx interface{}, cmdCtx *Context) { // 在消息事件触发时调用
	for alias, opts := range r.format.Alias {
		reg := regexp.MustCompile(alias)
		if reg.MatchString(cmdCtx.ArgumentString) { // 如果匹配指令别名
			cmdCtx.Command = r.format.Name
			matches := reg.FindStringSubmatch(cmdCtx.ArgumentString) // 正则中的子串
			for _, arg := range r.format.Arguments {                 // 遍历参数
				if val, ok := cmdCtx.Options[arg.Name]; ok {
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
			for _, opt := range r.format.Options { // 遍历选项
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
			r.handler(botCtx, cmdCtx)
			return
		}
	}
}

func (r *Registry) parse(cmd *Context) error {
	var reader io.Reader = strings.NewReader(cmd.ArgumentString)
	l := shlex.NewLexer(reader)
	reader = getReader(l)
	cmdName, err := l.Next()
	if err != nil {
		return err
	}
	if cmd.Command = cmdName; cmdName != r.format.Name {
		return fmt.Errorf("unmatched command")
	}

	var now string

	argIndex := 0
	for {
		if argIndex < len(r.format.Arguments) && r.format.Arguments[argIndex].Type == TextArg {
			text, err := io.ReadAll(reader)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("error occurred while parsing argument '%s'", r.format.Arguments[argIndex].Name)
			}
			if strings.HasPrefix(string(text), "-") {
				l = shlex.NewLexer(strings.NewReader(string(text)))
				reader = getReader(l)
			} else {
				cmd.Options[r.format.Arguments[argIndex].Name] = string(text)
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
			for _, opt := range r.format.Options {
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
		if argIndex < len(r.format.Arguments) {
			cmd.Args[r.format.Arguments[argIndex].Name] = now
			argIndex++
		} else {
			return fmt.Errorf("too many arguments")
		}
	}

	if err == nil || err == io.EOF {
		return nil
	} else {
		return fmt.Errorf("error occurred while parsing command '%v'", err)
	}
}

func (r *Registry) checkRequired(cmdCtx *Context) error {
	for _, arg := range r.format.Arguments {
		if _, ok := cmdCtx.Args[arg.Name]; !ok {
			if arg.Required {
				return fmt.Errorf("missing required argument '%s'", arg.Name)
			} else {
				cmdCtx.Args[arg.Name] = arg.Default
			}
		}

	}

	for _, opt := range r.format.Options {
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
