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

	"github.com/oussamasf/yuji/service/tcp"
	"github.com/oussamasf/yuji/utils"
)

func HandleReplicaConnection(masterHost string, masterPort string, replicaPort string, cache map[string]string) {
	address := fmt.Sprintf("%s:%s", masterHost, masterPort)
	m, err := net.Dial("tcp", address)
	if err != nil {
		log.Fatalln("couldn't connect to master at ", address)
	}

	defer m.Close()

	tcp.WriteArrayResp(m, []string{"ping"})
	time.Sleep(100 * time.Millisecond)

	tcp.WriteArrayResp(m, []string{"REPLCONF", "capa", "psync2"})
	time.Sleep(100 * time.Millisecond)

	tcp.WriteArrayResp(m, []string{"REPLCONF", "listening-port", replicaPort})
	time.Sleep(100 * time.Millisecond)

	tcp.WriteArrayResp(m, []string{"PSYNC", "?", "-1"})

	buffer := make([]byte, 1028)
	for {
		n, err := m.Read(buffer)
		if err != nil {
			if err == io.EOF {
				log.Println("Connection closed by master")
				break
			}
			log.Printf("Error reading from master: %v", err)
			break
		}

		if n > 0 {
			trimmedBuffer := bytes.Trim(buffer[:n], "\x00")

			formattedInput := strings.ReplaceAll(string(trimmedBuffer), "\\r\\n", "\r\n")

			commands, err := utils.Parser(formattedInput)

			if err != nil {
				if err == io.EOF {
					continue
				}
				continue
			}

			if args, ok := commands.Value.([]utils.RESPValue); ok {
				bytesCount := +len(formattedInput)

				cmdName, _ := args[0].Value.(string)

				switch strings.ToLower(cmdName) {
				case "replconf":
					tcp.WriteArrayResp(m, []string{"replconf", "ack", fmt.Sprint(bytesCount)})
				case "set":
					if len(args) < 3 {
						tcp.WriteRESPError(m, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
						continue
					}
					key, ok := args[1].Value.(string)
					if !ok {
						tcp.WriteRESPError(m, "ERROR: INVALID_ARGUMENT_TYPE")
						continue
					}
					value, ok := args[2].Value.(string)
					if !ok {
						tcp.WriteRESPError(m, "ERROR: INVALID_ARGUMENT_TYPE")
						continue
					}
					cache[key] = value

					if len(args) > 4 {

						if strings.ToLower(args[3].Value.(string)) == "px" {
							expiry, err := strconv.Atoi(args[4].Value.(string))
							if err != nil {
								tcp.WriteRESPError(m, "ERROR: INVALID_PX")
								continue
							}
							time.AfterFunc(time.Duration(expiry)*time.Millisecond, func() {
								delete(cache, key)
							})
						} else {
							tcp.WriteRESPError(m, "ERROR: INVALID_ARGUMENT")
							continue
						}
					}
					tcp.WriteRESPSimpleString(m, "OK")

				default:
					tcp.WriteResponse(m, "ERROR: Unknown command")
					return
				}
			}

		}
	}
}

func WriteCommandSync(replicaPorts []net.Conn, command []byte) {

	for _, m := range replicaPorts {
		m.Write(command)
	}
}
