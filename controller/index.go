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

	configuration "github.com/oussamasf/yuji/config"
	"github.com/oussamasf/yuji/service/tcp"
	"github.com/oussamasf/yuji/utils"
)

var replicasConnections = []net.Conn{}

var txQueue = configuration.TransactionSettings{
	InvokedTx: false,
}

func HandleConnection(conn net.Conn, config *configuration.AppSettings) {
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

		args, ok := commands.Value.([]configuration.RESPValue)
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
			if len(txQueue.Session) == 0 {
				tcp.WriteArrayResp(conn, []string{})
			} else {
				for _, session := range txQueue.Session {
					switch session.Cmd {

					case "set":
						res, err := handleSetCmd(session.Args, config.RedisMap)
						if err != nil {
							tcp.WriteRESPError(conn, fmt.Sprint(err))
						}

						tcp.WriteRESPSimpleString(conn, res)

					case "get":
						res, err := handleGetCmd(session.Args, config.RedisMap)
						if err != nil {
							tcp.WriteRESPError(conn, fmt.Sprint(err))
						}

						if res == "" {
							tcp.WriteRESPSimpleString(conn, res)
						} else {
							tcp.WriteRESPBulkString(conn, res)
						}

					case "incr":
						res, err := handleIncrCmd(session.Args, config.RedisMap)
						if err != nil {
							tcp.WriteRESPError(conn, fmt.Sprint(err))
						}

						if res == "" {
							tcp.WriteRESPSimpleString(conn, res)
						} else {
							tcp.WriteRESPBulkString(conn, res)
						}

					default:
						tcp.WriteArrayResp(conn, []string{})

					}
				}
			}

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
				handleIncrCmd(args, config.RedisMap)
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
			if txQueue.InvokedTx {
				session := configuration.TSession{
					Cmd:  strings.ToLower(cmdName),
					Args: args,
				}
				tcp.WriteRESPSimpleString(conn, "QUEUED")
				txQueue.Session = append(txQueue.Session, session)
				continue
			} else {
				handleSetCmd(args, config.RedisMap)
				// TODO support for replica in tx
				if !config.IsSlave {
					WriteCommandSync(replicasConnections, trimmedData)
				}
			}

		case "get":
			if txQueue.InvokedTx {
				session := configuration.TSession{
					Cmd:  strings.ToLower(cmdName),
					Args: args,
				}
				tcp.WriteRESPSimpleString(conn, "QUEUED")
				txQueue.Session = append(txQueue.Session, session)
				continue
			} else {
				handleGetCmd(args, config.RedisMap)
			}

		default:
			tcp.WriteResponse(conn, "ERROR: Unknown command")
			return
		}
	}
}

// ? GET
func handleGetCmd(args []configuration.RESPValue, cache map[string]string) (string, error) {

	if len(args) != 2 {
		return "", fmt.Errorf("ERROR: INVALID_NUMBER_OF_ARGUMENTS")
	}
	key, ok := args[1].Value.(string)
	if !ok {
		return "", fmt.Errorf("ERROR: INVALID_ARGUMENT_TYPE")
	}
	result := cache[key]
	return result, nil
}

// ? SET
func handleSetCmd(args []configuration.RESPValue, cache map[string]string) (string, error) {
	if len(args) < 3 {
		return "", fmt.Errorf("ERROR: INVALID_NUMBER_OF_ARGUMENTS")
	}

	key, ok := args[1].Value.(string)
	if !ok {
		return "", fmt.Errorf("ERROR: INVALID_ARGUMENT_TYPE")
	}

	value, ok := args[2].Value.(string)
	if !ok {
		return "", fmt.Errorf("ERROR: INVALID_ARGUMENT_TYPE")
	}

	cache[key] = value

	if len(args) > 4 {
		if strings.ToLower(args[3].Value.(string)) == "px" {
			expiry, err := strconv.Atoi(args[4].Value.(string))
			if err != nil {
				return "", fmt.Errorf("ERROR: INVALID_PX")
			}
			time.AfterFunc(time.Duration(expiry)*time.Millisecond, func() {
				delete(cache, key)
			})
		} else {
			return "", fmt.Errorf("ERROR: INVALID_ARGUMENT")
		}
	}

	return "OK", nil
}

func handleIncrCmd(args []configuration.RESPValue, cache map[string]string) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("ERROR: INVALID_NUMBER_OF_ARGUMENTS")
	}
	key, ok := args[1].Value.(string)
	if !ok {
		return "", fmt.Errorf("ERROR: INVALID_ARGUMENT_TYPE")
	}
	result, exists := cache[key]
	if !exists {
		cache[key] = "1"
	} else {
		intValue, err := strconv.Atoi(result)
		if err != nil {
			return "", fmt.Errorf("ERROR: CANNOT_INCR_NOT_INT")
		}
		cache[key] = strconv.Itoa(intValue + 1)
	}

	return cache[key], nil
}
