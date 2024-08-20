package utils

import (
	"fmt"
	"log"
	"net"
	"strconv"
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

func NewArrayResp(arr []string) string {
	if len(arr) == 0 {
		return "*0\r\n"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("*%d\r\n", len(arr)))

	for _, item := range arr {
		builder.WriteString(NewBulkString([]string{item}))
	}

	return builder.String()
}

func WriteArrayResp(conn net.Conn, arr []string) {
	if _, err := conn.Write([]byte(NewArrayResp(arr))); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func WriteRESPBulkString(conn net.Conn, message string) {
	response := fmt.Sprintf("$%d\r\n%s\r\n", len(message), message)
	if _, err := conn.Write([]byte(response)); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func WriteRESPSimpleString(conn net.Conn, message string) {
	response := fmt.Sprintf("+%s\r\n", message)
	if _, err := conn.Write([]byte(response)); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func WriteRESPError(conn net.Conn, message string) {
	response := fmt.Sprintf("-%s\r\n", message)
	if _, err := conn.Write([]byte(response)); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func WriteResponse(conn net.Conn, message string) {
	if _, err := conn.Write([]byte(message + "\n")); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func CompareIDs(id1, id2 string) int {
	parts1 := strings.Split(id1, "-")
	parts2 := strings.Split(id2, "-")

	if len(parts1) != 2 || len(parts2) != 2 {
		return 0 // Invalid format, treat as equal
	}

	timestamp1, err1 := strconv.ParseInt(parts1[0], 10, 64)
	timestamp2, err2 := strconv.ParseInt(parts2[0], 10, 64)
	if err1 != nil || err2 != nil {
		return 0
	}

	if timestamp1 != timestamp2 {
		if timestamp1 > timestamp2 {
			return 1
		}
		return -1
	}

	seq1, err1 := strconv.ParseInt(parts1[1], 10, 64)
	seq2, err2 := strconv.ParseInt(parts2[1], 10, 64)
	if err1 != nil || err2 != nil {
		return 0
	}

	if seq1 > seq2 {
		return 1
	} else if seq1 < seq2 {
		return -1
	}
	return 0
}
