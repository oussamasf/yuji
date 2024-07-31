package utils

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func writeResponse(conn net.Conn, message string) {
	if _, err := conn.Write([]byte(message + "\n")); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func HandleConnection(conn net.Conn, cache map[string]string, isSlave bool) {
	infoRes := []string{"role:master", "master_replid:8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb", "master_repl_offset:0"}

	defer conn.Close()

	data := make([]byte, 1028)
	for {
		n, err := conn.Read(data)
		if err != nil {
			log.Printf("Error parsing command: %v", err)
			writeResponse(conn, "ERROR: Invalid command")
			continue
		}

		trimmedData := bytes.TrimRight(data[:n], "\x00")
		commands, err := Parser(string(trimmedData))

		if err != nil {
			log.Printf("Error parsing command: %v", err)
			writeResponse(conn, "ERROR: Invalid command")
			continue
		}

		switch strings.ToLower(commands.Name) {
		case "info":
			if isSlave {
				infoRes = []string{"role:slave"}
			}
			writeResponse(conn, NewBulkString(infoRes))

		case "echo":
			if len(commands.Args) > 1 {
				writeResponse(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
				return
			}
			writeResponse(conn, commands.Args[0])

		case "ping":
			log.Printf("PONG")

			writeResponse(conn, "PONG")

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
						continue
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
			} else {
				writeResponse(conn, result)
			}

		default:
			writeResponse(conn, "ERROR: Unknown command")
			return
		}
	}
}
