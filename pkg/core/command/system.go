package command

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type System struct {
	Commands map[string]*Registry
	mu       sync.Mutex
}

func NewCommandSystem() *System {
	return &System{
		Commands: make(map[string]*Registry),
		mu:       sync.Mutex{},
	}
}

func (s *System) Register(registry Registry) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := uuid.New()
	copy := registry
	s.Commands[id.String()] = &copy
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.Commands, id.String())
	}
}

func (s *System) Emit(cmdCtx *Context) {
	s.mu.Lock()
	registries := make([]*Registry, 0, len(s.Commands))
	for _, registry := range s.Commands {
		registries = append(registries, registry)
	}
	s.mu.Unlock()

	for _, registry := range registries {
		ctx := cmdCtx.Clone()
		if err := registry.handle(ctx); err != nil && err.Error() != "unmatched command" {
			_, _ = ctx.ReplyText(err.Error())
		}
	}
}

func (r *Registry) Emit(cmdCtx *Context) error { // 触发指令Reg
	if r.Handler == nil {
		return fmt.Errorf("unmatched command")
	}
	return r.Handler(cmdCtx)
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
