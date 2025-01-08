package GoroBot

import (
	"database/sql"
	"fmt"
)

func (i *Instant) OpenDatabase(driverName string, dataSourceName string) error {
	var err error
	if i.db, err = sql.Open(driverName, dataSourceName); err != nil {
		i.db = nil
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	return nil
}

func (i *Instant) CloseDatabase() error {
	return i.db.Close()
}

func (i *Instant) Database() *sql.DB {
	return i.db
}

func (i *Instant) DatabaseExist() bool {
	return i.db != nil
}
