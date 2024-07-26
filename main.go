package main

import (
	"fmt"
	"net"
	"strings"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server is listening on :8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	_, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading:", err)
		return
	}

	words := strings.Fields(strings.ToLower(string(buffer)))

	switch words[0] {
	case "echo":
		_, err = conn.Write([]byte(strings.Join(words[1:], " ")))
		if err != nil {
			fmt.Println("Error writing:", err)
			return
		}
	default:
		fmt.Println("Error writing:", err)
	}

}
