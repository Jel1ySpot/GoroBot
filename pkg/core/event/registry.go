package event

import (
	"github.com/google/uuid"
	"sync"
)

type Registry struct {
	handlers map[string]*Handler
	mu       sync.Mutex
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]*Handler),
	}
}

func (r *Registry) append(f Callback) (*Handler, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := uuid.New().String()
	handler := &Handler{
		id:       id,
		callback: f,
		releaseFunc: func() {
			delete(r.handlers, id)
		},
	}
	r.handlers[handler.id] = handler
	return handler, nil
}

func (r *Registry) emit(args ...interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, handler := range r.handlers {
		if err := handler.call(args...); err != nil {
			return err
		}
	}
	return nil
}
