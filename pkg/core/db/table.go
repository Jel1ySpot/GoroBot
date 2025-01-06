package db

// TableField 字段信息
type TableField struct {
	Type     ObjectType
	Initial  any
	Nullable bool
}

// TableOption 表的配置
type TableOption struct {
	Fields        map[string]TableField
	PrimaryKey    string
	AutoIncrement bool
}
