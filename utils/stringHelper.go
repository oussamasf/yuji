package utils

import (
	"fmt"
	"log"
	"net"
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
