package controller

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	configuration "github.com/oussamasf/yuji/config"
	"github.com/oussamasf/yuji/service/tcp"
	"github.com/oussamasf/yuji/utils"
)

var replicasConnections = []net.Conn{}

var blockedStreamRequests = make(map[string][]*BlockedRequest)

type BlockedRequest struct {
	Conn       net.Conn
	StreamKeys []string
	Ids        []string
	BlockTime  time.Duration
	StartTime  time.Time
}

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

		cmdName, args, err := validateCommand(trimmedData)

		if err != nil {
			tcp.WriteRESPError(conn, err.Error())
			continue
		}

		switch cmdName {

		case "ping":
			tcp.WriteRESPSimpleString(conn, handlePingCmd())
		// case "save":
		// 	err := utils.SaveRDBFile(0, config)
		// 	if err != nil {
		// 		tcp.WriteRESPError(conn, "ERROR: COULD_NOT_SAVE_FILE")
		// 		continue
		// 	}
		// 	tcp.WriteRESPSimpleString(conn, "OK")
		case "type":
			key, err := handleTypeCmd(args)
			if err != nil {
				tcp.WriteRESPError(conn, err.Error())
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
			res, err := handleConfigCmd(args, config.Dir, config.DBFileName)

			if err != nil {
				tcp.WriteRESPError(conn, err.Error())
				continue
			}

			tcp.WriteArrayResp(conn, res)

		case "info":
			if config.IsSlave {
				infoRes = []string{"role:slave"}
			}
			tcp.WriteResponse(conn, utils.NewBulkString(infoRes))

		case "echo":
			res, err := handleEchoCmd(args)
			if err != nil {
				tcp.WriteRESPError(conn, err.Error())
				continue
			}
			tcp.WriteRESPBulkString(conn, res)

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

			streamKey, rawEntryID, keyValue, err := handleAddStreamCmd(args)

			if err != nil {
				tcp.WriteRESPError(conn, err.Error())
				continue
			}

			stream := configuration.IStream{
				Entries: []configuration.StreamEntry{},
			}

			newEntryID, err := utils.RefineRawID(rawEntryID, stream.LastID)
			if err != nil {
				tcp.WriteRESPError(conn, err.Error())
				continue
			}

			// ? Compare the new ID with the LastID in the stream
			if stream.LastID != "" && utils.CompareIDs(stream.LastID, newEntryID) >= 0 {
				tcp.WriteRESPError(conn, "ERROR: ERR The ID specified in XADD is equal or smaller than the target stream top item")
				continue
			}

			//? Check if the stream already exists in RedisMap
			if existingCache, found := config.RedisMap[streamKey]; found && existingCache.Type == configuration.Stream {
				stream = existingCache.StreamData
			}

			newEntry := configuration.StreamEntry{
				ID:     newEntryID,
				Values: keyValue,
			}

			//? Append the new stream entry
			stream.Entries = append(stream.Entries, configuration.StreamEntry{
				ID:     newEntryID,
				Values: keyValue,
			})

			stream.LastID = newEntryID

			//? Store the updated stream back in RedisMap
			config.RedisMap[streamKey] = configuration.ICache{
				Type:       configuration.Stream,
				StreamData: stream,
			}

			//? Check if any blocked XRead requests should be unblocked
			if blockedRequests, found := blockedStreamRequests[streamKey]; found {
				for _, request := range blockedRequests {
					//? Check if the new entry's ID is greater than the ID requested
					for _, requestedID := range request.Ids {
						if utils.CompareIDs(newEntryID, requestedID) > 0 {
							go func(conn net.Conn, streamKey, entryID string, entry configuration.StreamEntry) {
								values := []string{}
								for key, value := range entry.Values {
									values = append(values, fmt.Sprintf("$%d\r\n%s\r\n", len(key), key), fmt.Sprintf("$%d\r\n%s\r\n", len(value), value))
								}
								entryResp := fmt.Sprintf("*%d\r\n$%d\r\n%s\r\n*%d\r\n%s", 2, len(entryID), entryID, len(values)/2, strings.Join(values, ""))

								keyResp := fmt.Sprintf("*2\r\n$%d\r\n%s\r\n*%d\r\n%s", len(streamKey), streamKey, 1, entryResp)
								conn.Write([]byte(keyResp))
							}(request.Conn, streamKey, newEntryID, newEntry)

							break
						}
					}
				}
			}

			tcp.WriteRESPBulkString(conn, newEntryID)

		case "xread":

			ids, streamKeys, blockRequested, blockTime, nil := parseReadStreamArgs(args)
			if err != nil {
				tcp.WriteRESPError(conn, err.Error())
				continue
			}

			//? Ensure, have the same number of keys and IDs
			if len(streamKeys) != len(ids) {
				tcp.WriteRESPError(conn, "ERROR: MISMATCHED_KEYS_AND_IDS")
				continue
			}

			results := generateReadStreamResponse(ids, streamKeys, config)

			//? If results are found, send them immediately
			if len(results) > 0 {
				var builder strings.Builder
				builder.WriteString(fmt.Sprintf("*%d\r\n", len(results)))
				for _, result := range results {
					builder.WriteString(result)
				}

				conn.Write([]byte(builder.String()))
			}

			//? Handle blocking behavior
			if blockRequested {
				blockedRequest := &BlockedRequest{
					Conn:       conn,
					StreamKeys: streamKeys,
					Ids:        ids,
					BlockTime:  blockTime,
					StartTime:  time.Now(),
				}

				for _, streamKey := range streamKeys {
					blockedStreamRequests[streamKey] = append(blockedStreamRequests[streamKey], blockedRequest)
				}

				if blockTime > 0 {
					go func() {
						time.Sleep(blockTime)
						for _, streamKey := range streamKeys {
							if requests, found := blockedStreamRequests[streamKey]; found {
								for i, req := range requests {
									if req == blockedRequest {
										tcp.WriteRESPError(conn, "BLOCK timeout expired")
										blockedStreamRequests[streamKey] = append(requests[:i], requests[i+1:]...)
										break
									}
								}
							}
						}
					}()
				} else if blockTime == 0 {
					go func() {
						time.Sleep(time.Duration(1<<63 - 1))
					}()
				}
			} else {
				conn.Write([]byte("*0\r\n"))
			}

		case "xrange":
			if len(args) != 4 {
				tcp.WriteRESPError(conn, "ERROR: Invalid number of stream command arguments")
				continue
			}

			streamKey, ok := args[1].Value.(string)
			if !ok {
				tcp.WriteRESPError(conn, "ERROR: INVALID_ARGUMENT_TYPE")
				continue
			}
			stream, ok := config.RedisMap[streamKey]

			if !ok {
				tcp.WriteResponse(conn, "")
				continue
			}
			entries := stream.StreamData.Entries

			id1, _ := args[2].Value.(string)
			id2, _ := args[3].Value.(string)

			if utils.CompareIDs(id1, id2) > 0 {
				tcp.WriteRESPError(conn, "ERROR invalid range id")
				continue
			}
			results := []string{}
			for _, entry := range entries {
				if id2 == "+" {
					if utils.CompareIDs(entry.ID, id1) >= 0 {
						values := []string{}
						for key, value := range entry.Values {
							values = append(values, key, value)
						}

						respValues := utils.NewArrayResp(values)
						idResp := fmt.Sprintf("$%d\r\n%s\r\n", len(entry.ID), entry.ID)
						valueResp := fmt.Sprintf("*2\r\n%s\r\n%s\r\n", idResp, respValues)
						results = append(results, valueResp)
					}
					continue
				} else if id1 == "-" {
					if utils.CompareIDs(entry.ID, id2) <= 0 {
						values := []string{}
						for key, value := range entry.Values {
							values = append(values, key, value)
						}

						respValues := utils.NewArrayResp(values)
						idResp := fmt.Sprintf("$%d\r\n%s\r\n", len(entry.ID), entry.ID)
						valueResp := fmt.Sprintf("*2\r\n%s\r\n%s\r\n", idResp, respValues)
						results = append(results, valueResp)
					}
					continue
				} else if utils.CompareIDs(entry.ID, id1) >= 0 && utils.CompareIDs(entry.ID, id2) <= 0 {
					values := []string{}
					for key, value := range entry.Values {
						values = append(values, key, value)
					}

					respValues := utils.NewArrayResp(values)
					idResp := fmt.Sprintf("$%d\r\n%s\r\n", len(entry.ID), entry.ID)
					valueResp := fmt.Sprintf("*2\r\n%s\r\n%s\r\n", idResp, respValues)
					results = append(results, valueResp)
				}
			}
			var builder strings.Builder
			builder.WriteString(fmt.Sprintf("*%d\r\n", len(results)))

			for _, result := range results {
				builder.WriteString(result)
			}

			conn.Write([]byte(builder.String()))

		default:
			tcp.WriteRESPError(conn, "ERROR: Unknown command")
			return
		}
	}
}
