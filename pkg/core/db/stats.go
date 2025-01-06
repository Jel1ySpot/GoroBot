package db

type Stats struct {
	Size   uint64
	Tables map[string]TableStats
}

type TableStats struct {
	Size  uint64
	Count uint64
}
