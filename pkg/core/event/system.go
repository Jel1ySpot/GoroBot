package event

import (
	"fmt"
)

type System struct {
	events map[string]*Registry
}

func NewEventSystem() *System {
	return &System{
		events: make(map[string]*Registry),
	}
}

func (sys *System) Register(event string) {
	if _, ok := sys.events[event]; !ok {
		sys.events[event] = NewRegistry()
	}
}

func (sys *System) Unregister(event string) {
	delete(sys.events, event)
}

func (sys *System) On(event string, callback Callback) (func(), error) {
	if _, ok := sys.events[event]; !ok {
		return nil, fmt.Errorf("event not found")
	}
	handler, err := sys.events[event].append(callback)
	if err != nil {
		return nil, err
	}
	return handler.releaseFunc, nil
}

func (sys *System) Emit(event string, args ...interface{}) error {
	if _, ok := sys.events[event]; !ok {
		return fmt.Errorf("event %s not found", event)
	}
	return sys.events[event].emit(args...)
}
