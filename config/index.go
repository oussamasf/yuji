package configuration

type AppSettings struct {
	Port           string
	ReplicaAddress string
	Dir            string
	DBFileName     string
	IsSlave        bool
	RedisMap       map[string]ICache
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

type ICache struct {
	Data          string
	Type          CacheDataType
	ExpirationMap map[string]int64
}

type CacheDataType int

const (
	String CacheDataType = iota + 1
	Stream
	None
)

func (c CacheDataType) String() string {
	switch c {
	case String:
		return "String"
	case Stream:
		return "Stream"
	case None:
		return "None"
	default:
		return "Unknown"
	}
}
