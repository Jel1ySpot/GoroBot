package db

// Database 是一个数据库操作接口
type Database interface {
	// Connect 用于连接到数据库
	Connect(dsn string) error

	// Close 关闭数据库连接
	Close() error

	// Get 根据条件查询数据
	Get(table string, query Query) ([]ObjectRow, error)

	// Append 新增数据
	Append(table string, row ObjectRow) error

	// Create 创建表
	Create(table string, option TableOption) error

	// Drop 删除表
	Drop(table string) error

	// Stats 获取统计信息
	Stats() Stats
}
