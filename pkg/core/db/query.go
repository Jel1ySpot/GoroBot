package db

// FilterType 筛选类型
type FilterType int

const (
	NotField FilterType = iota
	OrField
	AndField
)

const (
	EqField FilterType = iota + 10
	NeField
	GtField
	GteField
	LtField
	LteField
)

const (
	HasField FilterType = iota + 20
	NHasField
)

const (
	RegexField FilterType = iota + 30
	RegexForField
)

// Query 查询表达式
type Query map[string]ObjectFilter

// ObjectFilter 筛选器
type ObjectFilter struct {
	parentQuery Query
	fieldType   FilterType
	value       any
}

func Which(key string) *ObjectFilter {
	query := make(Query)
	return query.Which(key)
}

func (q Query) Which(key string) *ObjectFilter {
	field := ObjectFilter{
		parentQuery: q,
	}
	q[key] = field
	return &field
}

func (f *ObjectFilter) Then() Query {
	return f.parentQuery
}

func (f *ObjectFilter) Not() *ObjectFilter {
	f.fieldType = NotField
	f.value = &ObjectFilter{
		parentQuery: f.parentQuery,
	}
	return f.value.(*ObjectFilter)
}

func (f *ObjectFilter) Or(val Query) *ObjectFilter {
	f.fieldType = OrField
	f.value = append(f.value.([]Query), val)
	return f
}

func (f *ObjectFilter) And(val Query) *ObjectFilter {
	f.fieldType = AndField
	f.value = append(f.value.([]Query), val)
	return f
}

func (f *ObjectFilter) Eq(value any) *ObjectFilter {
	f.fieldType = EqField
	f.value = value
	return f
}

func (f *ObjectFilter) Ne(value any) *ObjectFilter {
	f.fieldType = NeField
	f.value = value
	return f
}

func (f *ObjectFilter) Gt(value any) *ObjectFilter {
	f.fieldType = GtField
	f.value = value
	return f
}

func (f *ObjectFilter) Gte(value any) *ObjectFilter {
	f.fieldType = GteField
	f.value = value
	return f
}

func (f *ObjectFilter) Lt(value any) *ObjectFilter {
	f.fieldType = LtField
	f.value = value
	return f
}

func (f *ObjectFilter) Lte(value any) *ObjectFilter {
	f.fieldType = LteField
	f.value = value
	return f
}

func (f *ObjectFilter) Has(val any) *ObjectFilter {
	f.fieldType = HasField
	f.value = val
	return f
}

func (f *ObjectFilter) NotHas(val any) *ObjectFilter {
	f.fieldType = NHasField
	f.value = val
	return f
}

func (f *ObjectFilter) Match(val string) *ObjectFilter {
	f.fieldType = RegexField
	f.value = val
	return f
}

func (f *ObjectFilter) ToMatch(val string) *ObjectFilter {
	f.fieldType = RegexForField
	f.value = val
	return f
}

func (f *ObjectFilter) Build() Query {
	return f.parentQuery
}
