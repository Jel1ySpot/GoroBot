package message

type Builder struct {
	elements []*Element
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
