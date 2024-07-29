package utils

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func HandleConnection(conn net.Conn, cache map[string]string) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		data := scanner.Text()
		commands, err := Parser(data)
		if err != nil {
			log.Printf("Error parsing command: %v", err)
			writeResponse(conn, "ERROR: Invalid command")
			continue
		}

		switch strings.ToLower(commands.Name) {

		case "info":
			writeResponse(conn, NewBulkString("role:master"))

		case "echo":
			if len(commands.Args) > 2 {
				writeResponse(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
				return
			}

			writeResponse(conn, commands.Args[0])

		case "set":
			fmt.Println(commands.Args)

			if len(commands.Args) == 2 {
				cache[commands.Args[0]] = commands.Args[1]
				writeResponse(conn, "OK")

			} else if len(commands.Args) == 4 {

				if strings.ToLower(commands.Args[2]) == "px" {
					cache[commands.Args[0]] = commands.Args[1]

					parsedInt, err := strconv.Atoi(commands.Args[3])
					if err != nil {
						writeResponse(conn, "ERROR: INVALID_PX")

					}

					time.AfterFunc(time.Duration(parsedInt)*time.Millisecond, func() {
						delete(cache, commands.Args[0])
					})
					writeResponse(conn, "OK")

				} else {
					writeResponse(conn, "ERROR: INVALID_ARGUMENT")
					return
				}
			} else {
				writeResponse(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
				return
			}

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
