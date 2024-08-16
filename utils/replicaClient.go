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

func HandleReplicaConnection(masterHost string, masterPort string, replicaPort string, cache map[string]string) {
	address := fmt.Sprintf("%s:%s", masterHost, masterPort)
	m, err := net.Dial("tcp", address)
	if err != nil {
		log.Fatalln("couldn't connect to master at ", address)
	}

	defer m.Close()

	m.Write([]byte("*1\r\n$4\r\nping\r\n"))
	time.Sleep(100 * time.Millisecond)

	m.Write([]byte("*3\r\n$8\r\nREPLCONF\r\n$4\r\ncapa\r\n$6\r\npsync2\r\n"))
	time.Sleep(100 * time.Millisecond)

	m.Write([]byte("*3\r\n$5\r\nPSYNC\r\n$1\r\n?\r\n$2\r\n-1\r\n"))
	time.Sleep(100 * time.Millisecond)

	replConfig := fmt.Sprintf("*3\r\n$8\r\nREPLCONF\r\n$14\r\nlistening-port\r\n$4\r\n%s\r\n", replicaPort)
	m.Write([]byte(replConfig))

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
			trimmedBuffer := bytes.Trim(buffer[:n], "\x00\r\n")

			formattedInput := strings.ReplaceAll(string(trimmedBuffer), "\\r\\n", "\r\n")

			commands, err := Parser(formattedInput)

			if err != nil {
				if err == io.EOF {
					continue
				}
				log.Printf("Error parsing command: %v", err)
				WriteRESPError(m, "ERROR: Invalid command")
				continue
			}

			if args, ok := commands.Value.([]RESPValue); ok {
				cmdName, _ := args[0].Value.(string)
				switch strings.ToLower(cmdName) {
				case "set":
					if len(args) < 3 {
						WriteRESPError(m, "ERROR: INVALID_NUMBER_OF_ARGUMENTS")
						continue
					}
					key, ok := args[1].Value.(string)
					if !ok {
						WriteRESPError(m, "ERROR: INVALID_ARGUMENT_TYPE")
						continue
					}
					value, ok := args[2].Value.(string)
					if !ok {
						WriteRESPError(m, "ERROR: INVALID_ARGUMENT_TYPE")
						continue
					}
					cache[key] = value

					if len(args) > 4 {

						if strings.ToLower(args[3].Value.(string)) == "px" {
							expiry, err := strconv.Atoi(args[4].Value.(string))
							if err != nil {
								WriteRESPError(m, "ERROR: INVALID_PX")
								continue
							}
							time.AfterFunc(time.Duration(expiry)*time.Millisecond, func() {
								delete(cache, key)
							})
						} else {
							WriteRESPError(m, "ERROR: INVALID_ARGUMENT")
							continue
						}
					}
					WriteRESPSimpleString(m, "OK")

				default:
					WriteResponse(m, "ERROR: Unknown command")
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
