package configuration

type AppSettings struct {
	Port           string
	ReplicaAddress string
	Dir            string
	DBFileName     string
	IsSlave        bool
	RedisMap       map[string]string
	ExpirationMap  map[string]int64
}

type RESPValue struct {
	Type  byte
	Value interface{}
}

type TSession struct {
	Cmd  string
	Args []RESPValue
}

type TransactionSettings struct {
	InvokedTx bool
	Session   []TSession
}
