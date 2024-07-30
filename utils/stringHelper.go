package utils

import (
	"fmt"
	"strings"
)

const (
	NULL_BULK_STRING = "$-1\r\n"
	OK               = "OK"
)

func NewBulkString(arr []string) string {
	if len(arr) == 0 {
		return NULL_BULK_STRING
	}
	joinedStr := strings.Join(arr, "\r\n")
	return fmt.Sprintf("$%d\r\n%v\r\n", len(joinedStr), joinedStr)
}
