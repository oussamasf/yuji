package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/oussamasf/yuji/utils"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server is listening on :8080")
	redisMap := make(map[string]string)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go handleConnection(conn, redisMap)
	}
}

func handleConnection(conn net.Conn, cache map[string]string) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	_, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading:", err)
		return
	}

	data := string(buffer)

	data = strings.TrimSpace(data)

	commands, _ := utils.Parser(data)
	switch strings.ToLower(commands.Name) {
	case "echo":
		if len(commands.Args) > 2 {
			fmt.Println("INVALID_NUMBER_OF_ARGUMENTS")
			return
		}

		_, err = conn.Write([]byte(commands.Args[0] + "\n"))
		if err != nil {
			fmt.Println("Error writing:", err)
			return
		}
	case "set":

		if len(commands.Args) != 3 {
			fmt.Println("INVALID_NUMBER_OF_ARGUMENTS")
			return
		}
		cache[commands.Args[0]] = commands.Args[1]
		fmt.Println(cache)

	case "get":
		fmt.Println(cache)

		if len(commands.Args) != 2 {
			fmt.Println("INVALID_NUMBER_OF_ARGUMENTS")
			return
		}
		_, err = conn.Write([]byte(cache[commands.Args[0]] + "\n"))
		if err != nil {
			fmt.Println("Error writing:", err)
			return
		}

	default:
		fmt.Println("Error writing:", err)
	}

}
