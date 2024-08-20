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

func HandleConnection(conn net.Conn, config *configuration.AppSettings) {

	var txQueue = configuration.TransactionSettings{
		InvokedTx: false,
	}

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
		// case "save":
		// 	err := utils.SaveRDBFile(0, config)
		// 	if err != nil {
		// 		tcp.WriteRESPError(conn, "ERROR: COULD_NOT_SAVE_FILE")
		// 		continue
		// 	}
		// 	tcp.WriteRESPSimpleString(conn, "OK")
		case "type":

			if len(args) != 2 {
				tcp.WriteRESPError(conn, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
			}

			key, ok := args[1].Value.(string)
			if !ok {
				tcp.WriteRESPError(conn, "ERROR: MUST_PROVIDE_KEY")
				continue
			}

			tcp.WriteRESPSimpleString(conn, config.RedisMap[key].Type.String())

		case "multi":
			txQueue.InvokedTx = true
			tcp.WriteRESPSimpleString(conn, "OK")
		case "discard":
			if !txQueue.InvokedTx {
				tcp.WriteRESPError(conn, "ERROR: DISCARD without MULTI")
				continue
			}
			txQueue.Session = []configuration.TSession{}
			txQueue.InvokedTx = false
			tcp.WriteRESPSimpleString(conn, "OK")

		case "exec":
			if !txQueue.InvokedTx {
				tcp.WriteRESPError(conn, "ERROR: EXEC without MULTI")
				continue
			}
			if len(txQueue.Session) == 0 {
				tcp.WriteArrayResp(conn, []string{})
				txQueue.InvokedTx = false
			} else {
				results := []string{}
				for _, session := range txQueue.Session {
					switch session.Cmd {

					case "set":
						res, err := handleSetCmd(session.Args, config.RedisMap)
						if err != nil {
							results = append(results, fmt.Sprint(err))
							continue
						}
						results = append(results, res)

					case "get":
						res, err := handleGetCmd(session.Args, config.RedisMap)
						if err != nil {
							results = append(results, fmt.Sprint(err))
							continue
						}

						results = append(results, res)

					case "incr":
						res, err := handleIncrCmd(session.Args, config.RedisMap)
						if err != nil {
							results = append(results, fmt.Sprint(err))
							continue
						}

						results = append(results, res)

					default:
						tcp.WriteArrayResp(conn, []string{})
					}
				}

				tcp.WriteArrayResp(conn, results)
				txQueue.InvokedTx = false
			}

		case "incr":
			if txQueue.InvokedTx {
				session := configuration.TSession{
					Cmd:  strings.ToLower(cmdName),
					Args: args,
				}
				tcp.WriteRESPSimpleString(conn, "QUEUED")
				txQueue.Session = append(txQueue.Session, session)
				continue
			} else {

				res, err := handleIncrCmd(args, config.RedisMap)

				if err != nil {
					tcp.WriteRESPError(conn, "ERROR: DISCARD without MULTI")
					continue
				}

				tcp.WriteRESPBulkString(conn, res)

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

			//TODO send bulk string of hard coded empty RDB file after full resync
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
				_, err := handleSetCmd(args, config.RedisMap)

				if err != nil {
					tcp.WriteRESPError(conn, "ERROR: DISCARD without MULTI")
					continue
				}

				tcp.WriteRESPSimpleString(conn, "OK")

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

				res, err := handleGetCmd(args, config.RedisMap)

				if err != nil {
					tcp.WriteRESPError(conn, "ERROR: DISCARD without MULTI")
					continue
				}

				tcp.WriteRESPBulkString(conn, res)

				handleGetCmd(args, config.RedisMap)
			}

		case "xadd":
			stream := make(map[string][]string)

			//? check number of arguments
			if (len(args)%2 == 0) || (len(args) < 3) {
				tcp.WriteResponse(conn, "ERROR: Invalid number of stream command arguments")
				continue
			}

			//? cast stream key into string
			streamKey, ok := args[1].Value.(string)
			if !ok {
				tcp.WriteResponse(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}

			//? cast stream id into string
			id, ok := args[2].Value.(string)
			if !ok {
				tcp.WriteResponse(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}

			//? cast key-value pair of the steam into string array
			keyValue := []string{}
			for _, value := range args[3:] {
				castValue, ok := value.Value.(string)
				if !ok {
					tcp.WriteResponse(conn, "ERROR: unexpected error")
					continue
				}
				keyValue = append(keyValue, castValue)
			}

			//? if stream is not empty we overwrite the empty stream
			fmt.Println("config when virgin", config.RedisMap[streamKey].Data)
			if config.RedisMap[streamKey].Data != "" {
				stream, err = utils.DeserializeStream(config.RedisMap[streamKey].Data)
				if err != nil {
					tcp.WriteResponse(conn, "ERROR: deserialization error")
					continue
				}
			}
			stream[id] = keyValue
			fmt.Println("after stream ", stream)

			serializedStream, err := utils.SerializeStream(stream)
			if err != nil {
				tcp.WriteResponse(conn, "ERROR: serialization error")
				continue
			}

			config.RedisMap[streamKey] = configuration.ICache{
				Type: configuration.CacheDataType(2),
				Data: serializedStream,
			}

		default:
			tcp.WriteResponse(conn, "ERROR: Unknown command")
			return
		}
	}
}

// ? GET
func handleGetCmd(args []configuration.RESPValue, cache map[string]configuration.ICache) (string, error) {

	if len(args) != 2 {
		return "", fmt.Errorf("ERROR: INVALID_NUMBER_OF_ARGUMENTS")
	}
	key, ok := args[1].Value.(string)
	if !ok {
		return "", fmt.Errorf("ERROR: INVALID_ARGUMENT_TYPE")
	}
	result := cache[key]
	return result.Data, nil
}

// ? SET
func handleSetCmd(args []configuration.RESPValue, cache map[string]configuration.ICache) (string, error) {
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

	cache[key] = configuration.ICache{
		Data: value,
		Type: configuration.CacheDataType(1),
	}

	// TODO change this to store also px in db
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

func handleIncrCmd(args []configuration.RESPValue, cache map[string]configuration.ICache) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf("ERROR: INVALID_NUMBER_OF_ARGUMENTS")
	}
	key, ok := args[1].Value.(string)
	if !ok {
		return "", fmt.Errorf("ERROR: INVALID_ARGUMENT_TYPE")
	}
	result, exists := cache[key]
	if !exists {
		cache[key] = configuration.ICache{
			Data: "1",
		}
	} else {
		intValue, err := strconv.Atoi(result.Data)
		if err != nil {
			return "", fmt.Errorf("ERROR: CANNOT_INCR_NOT_INT")
		}
		cache[key] = configuration.ICache{
			Data: strconv.Itoa(intValue + 1),
		}

	}

	return cache[key].Data, nil
}
