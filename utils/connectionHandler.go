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

func HandleConnection(conn net.Conn, cache map[string]string, isSlave bool) {
	infoRes := []string{"role:master", "master_replid:8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb", "master_repl_offset:0"}

	defer conn.Close()

	data := make([]byte, 1028)
	for {
		n, err := conn.Read(data)
		if err != nil {
			log.Printf("Error parsing command: %v", err)
			WriteResponse(conn, "ERROR: Invalid command")
			continue
		}

		trimmedData := bytes.TrimRight(data[:n], "\x00")
		formattedInput := strings.ReplaceAll(string(trimmedData), "\\r\\n", "\r\n")
		commands, err := Parser(formattedInput)

		if err != nil {
			log.Printf("Error parsing command: %v", err)
			WriteRESPError(conn, "ERROR: Invalid command")
			continue
		}

		if commands.Type != '*' {
			WriteRESPError(conn, "ERROR: Expected array for command")
			continue
		}

		args, ok := commands.Value.([]RESPValue)
		if !ok {
			WriteRESPError(conn, "ERROR: Invalid command format")
			continue
		}

		if len(args) < 1 {
			WriteRESPError(conn, "ERROR: No command given")
			continue
		}

		cmdName, ok := args[0].Value.(string)
		if !ok {
			WriteRESPError(conn, "ERROR: Invalid command name")
			continue
		}

		switch strings.ToLower(cmdName) {
		case "info":
			if isSlave {
				infoRes = []string{"role:slave"}
			}
			WriteResponse(conn, NewBulkString(infoRes))

		case "echo":
			if len(args) != 2 {
				WriteRESPError(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
				continue
			}
			msg, ok := args[1].Value.(string)
			if !ok {
				WriteRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}
			WriteRESPBulkString(conn, msg)

		case "replconf":
			log.Printf("replconf")
			WriteRESPSimpleString(conn, "OK")

		case "psync":
			log.Printf("replconf")
			hardCoddedId := "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb"

			WriteRESPSimpleString(conn, fmt.Sprintf("FULLRESYNC %s 0", hardCoddedId))

			//? send bulk string of hard coded empty RDB file after full resync
			emptyRDB := "524544495330303131fa0972656469732d76657205372e322e30fa0a72656469732d62697473c040fa056374696d65c26d08bc65fa08757365642d6d656dc2b0c41000fa08616f662d62617365c000fff06e3bfec0ff5aa2"
			WriteRESPBulkString(conn, emptyRDB)

		case "ping":
			log.Printf("PONG")
			WriteRESPSimpleString(conn, "PONG")

		case "set":
			if len(args) < 3 {
				WriteRESPError(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
				continue
			}
			key, ok := args[1].Value.(string)
			if !ok {
				WriteRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}
			value, ok := args[2].Value.(string)
			if !ok {
				WriteRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}
			cache[key] = value

			if len(args) > 4 {

				if strings.ToLower(args[3].Value.(string)) == "px" {
					expiry, err := strconv.Atoi(args[4].Value.(string))
					if err != nil {
						WriteRESPError(conn, "ERROR: INVALID_PX")
						continue
					}
					time.AfterFunc(time.Duration(expiry)*time.Millisecond, func() {
						delete(cache, key)
					})
				} else {
					WriteRESPError(conn, "ERROR: INVALID_ARGUMENT")
					continue
				}
			}
			WriteRESPSimpleString(conn, "OK")

		case "get":
			if len(args) != 2 {
				WriteRESPError(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
				continue
			}
			key, ok := args[1].Value.(string)
			if !ok {
				WriteRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}
			result := cache[key]
			if result == "" {
				WriteRESPBulkString(conn, "")
			} else {
				WriteRESPBulkString(conn, result)
			}

		default:
			WriteResponse(conn, "ERROR: Unknown command")
			return
		}
	}
}

// SET
