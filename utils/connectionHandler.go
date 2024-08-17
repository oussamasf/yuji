package utils

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

var replicasConnections = []net.Conn{}

type Config struct {
	Port          string
	ReplicaType   string
	Dir           string
	DBFileName    string
	IsSlave       bool
	RedisMap      map[string]string
	ExpirationMap map[string]int64
}

type TX struct {
	InvokedTx bool
}

var txQueue = TX{
	InvokedTx: false,
}

func HandleConnection(conn net.Conn, config *Config) {
	infoRes := []string{"role:master", "master_replid:8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb", "master_repl_offset:0"}

	defer conn.Close()

	data := make([]byte, 1028)
	for {
		n, err := conn.Read(data)
		if err != nil {
			if err == io.EOF {
				log.Println("Connection closed")
				break
			}
			log.Printf("Error reading: %v", err)
			break
		}
		trimmedData := bytes.TrimRight(data[:n], "\x00")
		formattedInput := strings.ReplaceAll(string(trimmedData), "\\r\\n", "\r\n")

		commands, err := Parser(formattedInput)

		if err != nil {
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
		case "save":
			err := SaveRDBFile(0, config)
			if err != nil {
				WriteRESPError(conn, "ERROR: COULD_NOT_SAVE_FILE")
				continue
			}
			WriteRESPSimpleString(conn, "OK")
		case "multi":
			txQueue = TX{
				InvokedTx: true,
			}
			WriteRESPSimpleString(conn, "OK")

		case "exec":
			if !txQueue.InvokedTx {
				WriteRESPError(conn, "ERROR: EXEC without MULTI")
				continue
			}

			WriteArrayResp(conn, []string{})

		case "incr":
			if len(args) != 2 {
				WriteRESPError(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
				continue
			}
			key, ok := args[1].Value.(string)
			if !ok {
				WriteRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}
			result, exists := config.RedisMap[key]
			if !exists {
				config.RedisMap[key] = "1"
			} else {
				intValue, err := strconv.Atoi(result)
				if err != nil {
					WriteRESPError(conn, "ERROR: CANNOT_INCR_NOT_INT")
					continue
				}
				config.RedisMap[key] = strconv.Itoa(intValue + 1)
			}

			if txQueue.InvokedTx {
				WriteRESPSimpleString(conn, "QUEUED")
				continue
			} else {
				WriteRESPBulkString(conn, config.RedisMap[key])
			}

		case "keys":
			keys, err := LogFileKeys()
			if err != nil {
				WriteRESPError(conn, "ERROR: PARSE_ERROR")
				continue
			}
			WriteArrayResp(conn, keys)

		case "config":
			subcommand, ok := args[1].Value.(string)
			subcommand = strings.ToLower(subcommand)
			if !ok && subcommand != "get" {
				WriteRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}

			value, ok := args[2].Value.(string)
			value = strings.ToLower(value)

			if !ok {
				WriteRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}

			if value == "dir" {
				WriteArrayResp(conn, []string{"dir", fmt.Sprint(len(config.Dir)), config.Dir})
			} else if value == "dbfilename" {
				WriteArrayResp(conn, []string{"dbfilename", fmt.Sprint(len(config.DBFileName)), config.DBFileName})
			} else {
				WriteRESPError(conn, "ERR unsupported CONFIG parameter")
			}

		case "info":
			if config.IsSlave {
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
			WriteRESPSimpleString(conn, "OK")

		case "psync":
			hardCoddedId := "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb"

			WriteRESPSimpleString(conn, fmt.Sprintf("FULLRESYNC %s 0", hardCoddedId))
			time.Sleep(100 * time.Millisecond)

			//? send bulk string of hard coded empty RDB file after full resync
			emptyRDB := "524544495330303131fa0972656469732d76657205372e322e30fa0a72656469732d62697473c040fa056374696d65c26d08bc65fa08757365642d6d656dc2b0c41000fa08616f662d62617365c000fff06e3bfec0ff5aa2"

			WriteRESPBulkString(conn, emptyRDB)
			time.Sleep(100 * time.Millisecond)

			WriteArrayResp(conn, []string{"replconf", "getack", "*"})

			replicasConnections = append(replicasConnections, conn)

		case "ping":
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
			config.RedisMap[key] = value

			if len(args) > 4 {

				if strings.ToLower(args[3].Value.(string)) == "px" {
					expiry, err := strconv.Atoi(args[4].Value.(string))
					if err != nil {
						WriteRESPError(conn, "ERROR: INVALID_PX")
						continue
					}
					time.AfterFunc(time.Duration(expiry)*time.Millisecond, func() {
						delete(config.RedisMap, key)
					})
				} else {
					WriteRESPError(conn, "ERROR: INVALID_ARGUMENT")
					continue
				}
			}

			if txQueue.InvokedTx {
				WriteRESPSimpleString(conn, "QUEUED")
				continue
			} else {
				WriteRESPSimpleString(conn, "OK")
			}

			if !config.IsSlave {
				WriteCommandSync(replicasConnections, trimmedData)
			}

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
			result := config.RedisMap[key]
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
