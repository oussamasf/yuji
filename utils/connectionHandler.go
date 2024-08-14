package utils

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strings"
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

		formattedInput := strings.ReplaceAll(string(trimmedData), "\\r\\n", "\r\n")

		commands, err := Parser(formattedInput)

		if err != nil {
			log.Printf("Error parsing command: %v", err)
			writeRESPError(conn, "ERROR: Invalid command")
			continue
		}

		if commands.Type != '*' {
			writeRESPError(conn, "ERROR: Expected array for command")
			continue
		}

		args, ok := commands.Value.([]RESPValue)
		if !ok {
			writeRESPError(conn, "ERROR: Invalid command format")
			continue
		}

		if len(args) < 1 {
			writeRESPError(conn, "ERROR: No command given")
			continue
		}

		cmdName, ok := args[0].Value.(string)
		if !ok {
			writeRESPError(conn, "ERROR: Invalid command name")
			continue
		}

		switch strings.ToLower(cmdName) {
		case "info":
			if isSlave {
				infoRes = []string{"role:slave"}
			}
			writeResponse(conn, NewBulkString(infoRes))

		case "echo":
			if len(args) != 2 {
				writeRESPError(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
				continue
			}
			msg, ok := args[1].Value.(string)
			if !ok {
				writeRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}
			writeRESPBulkString(conn, msg)

		case "ping":
			log.Printf("PONG")
			writeRESPSimpleString(conn, "PONG")

			// case "set":
			// 	fmt.Println(commands.Args)
			// 	if len(commands.Args) == 2 {
			// 		cache[commands.Args[0]] = commands.Args[1]
			// 		writeResponse(conn, "OK")
			// 	} else if len(commands.Args) == 4 {
			// 		if strings.ToLower(commands.Args[2]) == "px" {
			// 			cache[commands.Args[0]] = commands.Args[1]
			// 			parsedInt, err := strconv.Atoi(commands.Args[3])
			// 			if err != nil {
			// 				writeResponse(conn, "ERROR: INVALID_PX")
			// 				continue
			// 			}
			// 			time.AfterFunc(time.Duration(parsedInt)*time.Millisecond, func() {
			// 				delete(cache, commands.Args[0])
			// 			})
			// 			writeResponse(conn, "OK")
			// 		} else {
			// 			writeResponse(conn, "ERROR: INVALID_ARGUMENT")
			// 			return
			// 		}
			// 	} else {
			// 		writeResponse(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
			// 		return
			// 	}

			// case "get":
			// fmt.Println(cache)
			// if len(commands.Args) > 1 {
			// 	writeResponse(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
			// 	return
			// }
			// result := cache[commands.Args[0]]
			// if result == "" {
			// 	writeResponse(conn, "NULL")
			// } else {
			// 	writeResponse(conn, result)
			// }

		default:
			writeResponse(conn, "ERROR: Unknown command")
			return
		}
	}
}

func writeRESPBulkString(conn net.Conn, message string) {
	response := fmt.Sprintf("$%d\r\n%s\r\n", len(message), message)
	if _, err := conn.Write([]byte(response)); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func writeRESPSimpleString(conn net.Conn, message string) {
	response := fmt.Sprintf("+%s\r\n", message)
	if _, err := conn.Write([]byte(response)); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func writeRESPError(conn net.Conn, message string) {
	response := fmt.Sprintf("-%s\r\n", message)
	if _, err := conn.Write([]byte(response)); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}
