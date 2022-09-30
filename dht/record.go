package dht

type Record struct {
	Key     int
	OrigKey string
	Value   string
}

func NewRecord(key string, value string, keyspaceSize int) *Record {
	r := Record{HashString(key, keyspaceSize), key, value}
	return &r
}
