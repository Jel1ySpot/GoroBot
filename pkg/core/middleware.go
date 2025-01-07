package GoroBot

import (
	"github.com/Jel1ySpot/GoroBot/pkg/core/message"
	"github.com/google/uuid"
	"sync"
)

type MiddlewareCallback func(bot BotContext, msg message.Context, next func(...MiddlewareCallback) error) error

type MiddlewareSystem struct {
	middlewares map[string]MiddlewareCallback
	sortedIDs   []string
	mu          sync.Mutex
}

type Middleware struct {
	prepare  bool
	function MiddlewareCallback
}

func (i *Instant) Middleware(callback MiddlewareCallback, prepare ...bool) func() {
	if len(prepare) > 0 && prepare[0] {
		return i.middleware.add(callback, true)
	} else {
		return i.middleware.add(callback, false)
	}
}

func (sys *MiddlewareSystem) add(callback MiddlewareCallback, prepare bool) func() {
	sys.mu.Lock()
	defer sys.mu.Unlock()

	id := uuid.New().String()
	sys.middlewares[id] = callback
	if prepare {
		sys.sortedIDs = append([]string{id}, sys.sortedIDs...)
	} else {
		sys.sortedIDs = append(sys.sortedIDs, id)
	}
	return func() {
		sys.mu.Lock()
		defer sys.mu.Unlock()

		if _, ok := sys.middlewares[id]; ok {
			delete(sys.middlewares, id)
		}
	}
}

func (sys *MiddlewareSystem) dispatch(bot BotContext, msg message.Context, fn func() error) error {
	sys.mu.Lock()
	defer sys.mu.Unlock()

	index := 0

	var final []MiddlewareCallback
	var finalCallback func(cb ...MiddlewareCallback) error
	finalCallback = func(cb ...MiddlewareCallback) error {
		if len(cb) > 0 {
			final = append(final, cb...)
		}

		if index >= len(final) {
			return fn()
		}

		index++
		return final[index-1](bot, msg, finalCallback)
	}

	var callback func(cb ...MiddlewareCallback) error
	callback = func(cb ...MiddlewareCallback) error {
		if len(cb) > 0 {
			final = append(final, cb...)
		}

		if index >= len(sys.sortedIDs) {
			index = 0
			return finalCallback()
		}

		if m, ok := sys.middlewares[sys.sortedIDs[index]]; ok {
			index++
			return m(bot, msg, callback)
		} else {
			sys.sortedIDs = append(sys.sortedIDs[:index], sys.sortedIDs[index+1:]...)
			return callback()
		}
	}

	return callback()
}
