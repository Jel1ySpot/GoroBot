package message

type Context interface {
	Protocol() string
	String() string
	Message() *Base
	Reply(message []*Element) error
	ReplyText(text string) error
}
