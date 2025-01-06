package message

type Element struct {
	Type    ElementType
	Content string
	Source  string
}

type ElementType int

const (
	Text ElementType = iota
	Quote
	Mention
	Image
	Video
	File
	Voice
	Sticker
	Link
	Other
)
