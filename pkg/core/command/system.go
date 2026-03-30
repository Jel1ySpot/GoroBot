package command

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type System struct {
	commands map[string]*Registry
	mu       sync.RWMutex
}

func NewCommandSystem() *System {
	return &System{
		commands: make(map[string]*Registry),
	}
}

func (s *System) Register(registry Registry) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := uuid.New()
	copy := registry
	s.commands[id.String()] = &copy
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.commands, id.String())
	}
}

func (s *System) Emit(cmdCtx *Context) {
	s.mu.RLock()
	registries := make([]*Registry, 0, len(s.commands))
	for _, registry := range s.commands {
		registries = append(registries, registry)
	}
	s.mu.RUnlock()

	for _, registry := range registries {
		ctx := cmdCtx.Clone()
		if err := registry.handle(ctx); err != nil && err.Error() != "unmatched command" {
			_, _ = ctx.ReplyText(err.Error())
		}
	}
}

// GetSchemas 返回所有已注册的顶级命令 Schema
func (s *System) GetSchemas() []Schema {
	s.mu.RLock()
	defer s.mu.RUnlock()
	schemas := make([]Schema, 0, len(s.commands))
	for _, reg := range s.commands {
		schemas = append(schemas, reg.Schema)
	}
	return schemas
}

// CheckAliases 遍历所有已注册命令检查别名匹配
func (s *System) CheckAliases(ctx *Context) {
	s.mu.RLock()
	registries := make([]*Registry, 0, len(s.commands))
	for _, reg := range s.commands {
		registries = append(registries, reg)
	}
	s.mu.RUnlock()

	for _, reg := range registries {
		reg.CheckAlias(ctx.Clone())
	}
}

func (r *Registry) Emit(cmdCtx *Context) error { // 触发指令Reg
	if r.Handler == nil {
		return fmt.Errorf("unmatched command")
	}
	return r.Handler(cmdCtx.setSchema(&r.Schema))
}

func (r *Registry) handle(cmdCtx *Context) error {
	if err := cmdCtx.processTokens(&r.Schema); err != nil {
		return err
	}

	target := r.findRegistry(cmdCtx.Commands)
	if target == nil {
		return fmt.Errorf("unmatched command")
	}

	return target.Emit(cmdCtx)
}

func (r *Registry) findRegistry(commands []string) *Registry {
	if len(commands) == 0 || !r.Schema.Match(commands[0]) {
		return nil
	}

	if len(commands) == 1 {
		return r
	}

	for i := range r.SubRegistries {
		if sub := r.SubRegistries[i].findRegistry(commands[1:]); sub != nil {
			return sub
		}
	}

	return nil
}
