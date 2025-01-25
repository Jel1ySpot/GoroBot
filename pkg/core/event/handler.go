package event

type Callback func(args ...interface{})

type Handler struct {
	id          string
	callback    Callback
	releaseFunc func()
}

func (h *Handler) call(args ...interface{}) {
	go h.callback(args...)
}
