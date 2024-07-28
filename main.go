package main

import (
	"bufio"
	"fmt"
	"log"
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
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		data := scanner.Text()
		commands, err := utils.Parser(data)
		if err != nil {
			log.Printf("Error parsing command: %v", err)
			writeResponse(conn, "ERROR: Invalid command")
			continue
		}

		switch strings.ToLower(commands.Name) {

		case "echo":
			if len(commands.Args) > 2 {
				writeResponse(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
				return
			}

			writeResponse(conn, commands.Args[0])

		case "set":
			if len(commands.Args) > 2 {
				writeResponse(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
				return
			}
			cache[commands.Args[0]] = commands.Args[1]
			writeResponse(conn, "OK")

		case "get":
			fmt.Println(cache)

			if len(commands.Args) > 1 {
				writeResponse(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
				return
			}

			result := cache[commands.Args[0]]

			if result == "" {
				writeResponse(conn, "NULL")
			}

			writeResponse(conn, cache[commands.Args[0]])

		default:
			writeResponse(conn, "ERROR: Unknown command")
			return
		}
	}
}

func writeResponse(conn net.Conn, message string) {
	if _, err := conn.Write([]byte(message + "\n")); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}
