package tcp

import (
	"fmt"
	"log"
	"net"

	"github.com/oussamasf/yuji/utils"
)

func WriteArrayResp(conn net.Conn, arr []string) {
	if _, err := conn.Write([]byte(utils.NewArrayResp(arr))); err != nil {
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
