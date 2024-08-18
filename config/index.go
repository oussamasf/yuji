package config

type AppSettings struct {
	Port           string
	ReplicaAddress string
	Dir            string
	DBFileName     string
	IsSlave        bool
	RedisMap       map[string]string
	ExpirationMap  map[string]int64
}

type TransactionsSettings struct {
	InvokedTx bool
}
