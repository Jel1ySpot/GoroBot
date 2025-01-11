package message

import (
	"fmt"
)

type Builder struct {
	elements []*Element
}

func (b *Builder) Text(text string) *Builder {
	return b.Append(TextElement, text, "")
}

func (b *Builder) Quote(msg *Base) *Builder {
	return b.Append(QuoteElement, "[回复]", msg.Marshall())
}

func (b *Builder) Mention(id string) *Builder {
	return b.Append(MentionElement, fmt.Sprintf("@%s", id), id)
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Append(elementType ElementType, content string, source string) *Builder {
	b.elements = append(b.elements, &Element{
		Type:    elementType,
		Content: content,
		Source:  source,
	})
	return b
}

func (b *Builder) Build() []*Element {
	return b.elements
}
