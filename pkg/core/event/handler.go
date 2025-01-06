package event

type Callback func(args ...interface{}) error

type Handler struct {
	id          string
	callback    Callback
	releaseFunc func()
}

func (h *Handler) call(args ...interface{}) error {
	return h.callback(args...)
}
