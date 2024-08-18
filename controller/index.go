package controller

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/oussamasf/yuji/config"
	"github.com/oussamasf/yuji/service/tcp"
	"github.com/oussamasf/yuji/utils"
)

var replicasConnections = []net.Conn{}

var txQueue = config.TransactionsSettings{
	InvokedTx: false,
}

func HandleConnection(conn net.Conn, config *config.AppSettings) {
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

		commands, err := utils.Parser(formattedInput)

		if err != nil {
			tcp.WriteRESPError(conn, "ERROR: Invalid command")
			continue
		}

		if commands.Type != '*' {
			tcp.WriteRESPError(conn, "ERROR: Expected array for command")
			continue
		}

		args, ok := commands.Value.([]utils.RESPValue)
		if !ok {
			tcp.WriteRESPError(conn, "ERROR: Invalid command format")
			continue
		}

		if len(args) < 1 {
			tcp.WriteRESPError(conn, "ERROR: No command given")
			continue
		}

		cmdName, ok := args[0].Value.(string)
		if !ok {
			tcp.WriteRESPError(conn, "ERROR: Invalid command name")
			continue
		}

		switch strings.ToLower(cmdName) {
		case "save":
			err := utils.SaveRDBFile(0, config)
			if err != nil {
				tcp.WriteRESPError(conn, "ERROR: COULD_NOT_SAVE_FILE")
				continue
			}
			tcp.WriteRESPSimpleString(conn, "OK")
		case "multi":
			txQueue.InvokedTx = true
			tcp.WriteRESPSimpleString(conn, "OK")

		case "exec":
			if !txQueue.InvokedTx {
				tcp.WriteRESPError(conn, "ERROR: EXEC without MULTI")
				continue
			}

			tcp.WriteArrayResp(conn, []string{})

		case "incr":
			if len(args) != 2 {
				tcp.WriteRESPError(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
				continue
			}
			key, ok := args[1].Value.(string)
			if !ok {
				tcp.WriteRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}
			result, exists := config.RedisMap[key]
			if !exists {
				config.RedisMap[key] = "1"
			} else {
				intValue, err := strconv.Atoi(result)
				if err != nil {
					tcp.WriteRESPError(conn, "ERROR: CANNOT_INCR_NOT_INT")
					continue
				}
				config.RedisMap[key] = strconv.Itoa(intValue + 1)
			}

			if txQueue.InvokedTx {
				tcp.WriteRESPSimpleString(conn, "QUEUED")
				continue
			} else {
				tcp.WriteRESPBulkString(conn, config.RedisMap[key])
			}

		case "keys":
			keys, err := utils.LogFileKeys()
			if err != nil {
				tcp.WriteRESPError(conn, "ERROR: PARSE_ERROR")
				continue
			}
			tcp.WriteArrayResp(conn, keys)

		case "config":
			subcommand, ok := args[1].Value.(string)
			subcommand = strings.ToLower(subcommand)
			if !ok && subcommand != "get" {
				tcp.WriteRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}

			value, ok := args[2].Value.(string)
			value = strings.ToLower(value)

			if !ok {
				tcp.WriteRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}

			if value == "dir" {
				tcp.WriteArrayResp(conn, []string{"dir", fmt.Sprint(len(config.Dir)), config.Dir})
			} else if value == "dbfilename" {
				tcp.WriteArrayResp(conn, []string{"dbfilename", fmt.Sprint(len(config.DBFileName)), config.DBFileName})
			} else {
				tcp.WriteRESPError(conn, "ERR unsupported CONFIG parameter")
			}

		case "info":
			if config.IsSlave {
				infoRes = []string{"role:slave"}
			}
			tcp.WriteResponse(conn, utils.NewBulkString(infoRes))

		case "echo":
			if len(args) != 2 {
				tcp.WriteRESPError(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
				continue
			}
			msg, ok := args[1].Value.(string)
			if !ok {
				tcp.WriteRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}
			tcp.WriteRESPBulkString(conn, msg)

		case "replconf":
			tcp.WriteRESPSimpleString(conn, "OK")

		case "psync":
			hardCoddedId := "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb"

			tcp.WriteRESPSimpleString(conn, fmt.Sprintf("FULLRESYNC %s 0", hardCoddedId))
			time.Sleep(100 * time.Millisecond)

			//? send bulk string of hard coded empty RDB file after full resync
			emptyRDB := "524544495330303131fa0972656469732d76657205372e322e30fa0a72656469732d62697473c040fa056374696d65c26d08bc65fa08757365642d6d656dc2b0c41000fa08616f662d62617365c000fff06e3bfec0ff5aa2"

			tcp.WriteRESPBulkString(conn, emptyRDB)
			time.Sleep(100 * time.Millisecond)

			tcp.WriteArrayResp(conn, []string{"replconf", "getack", "*"})

			replicasConnections = append(replicasConnections, conn)

		case "ping":
			tcp.WriteRESPSimpleString(conn, "PONG")

		case "set":
			if len(args) < 3 {
				tcp.WriteRESPError(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
				continue
			}
			key, ok := args[1].Value.(string)
			if !ok {
				tcp.WriteRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}
			value, ok := args[2].Value.(string)
			if !ok {
				tcp.WriteRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}
			config.RedisMap[key] = value

			if len(args) > 4 {

				if strings.ToLower(args[3].Value.(string)) == "px" {
					expiry, err := strconv.Atoi(args[4].Value.(string))
					if err != nil {
						tcp.WriteRESPError(conn, "ERROR: INVALID_PX")
						continue
					}
					time.AfterFunc(time.Duration(expiry)*time.Millisecond, func() {
						delete(config.RedisMap, key)
					})
				} else {
					tcp.WriteRESPError(conn, "ERROR: INVALID_ARGUMENT")
					continue
				}
			}

			if txQueue.InvokedTx {
				tcp.WriteRESPSimpleString(conn, "QUEUED")
				continue
			} else {
				tcp.WriteRESPSimpleString(conn, "OK")
			}

			if !config.IsSlave {
				WriteCommandSync(replicasConnections, trimmedData)
			}

		case "get":
			if len(args) != 2 {
				tcp.WriteRESPError(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
				continue
			}
			key, ok := args[1].Value.(string)
			if !ok {
				tcp.WriteRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}
			result := config.RedisMap[key]
			if result == "" {
				tcp.WriteRESPBulkString(conn, "")
			} else {
				tcp.WriteRESPBulkString(conn, result)
			}

		default:
			tcp.WriteResponse(conn, "ERROR: Unknown command")
			return
		}
	}
}

// SET
