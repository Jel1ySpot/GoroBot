package message

type Context interface {
	String() string
	Message() *Base
	Protocol() string
	Reply(message []*Element) error
	ReplyText(text string) error
}
