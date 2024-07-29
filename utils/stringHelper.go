package utils

import "fmt"

const (
	NULL_BULK_STRING = "$-1\r\n"
	OK               = "OK"
)

func NewBulkString(str string) string {
	if str == "" {
		return NULL_BULK_STRING
	}
	return fmt.Sprintf("$%d\r\n%v\r\n", len(str), str)
}
