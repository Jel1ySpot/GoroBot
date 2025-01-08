# 数据库操作
GoroBot 实例中存放着一个数据库接口。暴露了几个方法用于简易连接与操作。

### grb.OpenDatabase(driverName string, dataSourceName string) error
打开数据库。同 `sql.Open(driverName, dataSourceName)`。

### grb.CloseDatabase() error
关闭数据库。

### grb.Database() *sql.DB
获取数据库实例。

### grb.DatabaseExist() bool
如果连接了数据库，返回 `true`，否则返回 `false`。
