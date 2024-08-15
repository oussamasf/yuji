package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/oussamasf/yuji/utils"
)

func main() {
	var r string
	var RSlice []string
	var isSlave = false

	port := flag.String("p", "8080", "port")
	replicaType := flag.String("replicaof", "", "replica of")
	flag.Parse()

	if *replicaType != "" {
		r = strings.TrimSpace(*replicaType)
		RSlice = strings.Split(r, ":")

		if len(RSlice) != 2 {
			fmt.Println("INVALID_REPLICA_ARGUMENT")
			return
		}

		if RSlice[1] == *port {
			fmt.Println("PORT_OF_REPLICA_SHOULD_BE_DIFFERENT_OF_MASTER")
			return
		}
		isSlave = true

		go SendHandshake(RSlice[0], RSlice[1], *port)
	}

	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server is listening on " + ":" + *port)

	redisMap := make(map[string]string)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go utils.HandleConnection(conn, redisMap, isSlave)
	}
}

func SendHandshake(masterHost string, masterPort string, replicaPort string) {
	address := fmt.Sprintf("%s:%s", masterHost, masterPort)
	m, err := net.Dial("tcp", address)
	if err != nil {
		log.Fatalln("couldn't connect to master at ", address)
	}
	m.Write([]byte("*1\r\n$4\r\nping\r\n"))

	buffer := make([]byte, 1028)

	n, err := m.Read(buffer)
	if err != nil {
		log.Fatalln("couldn't receive the pong")
	}
	trimmedBuffer := bytes.Trim(buffer[:n], "\x00\r\n")
	response := string(trimmedBuffer)
	fmt.Printf("Response: %q\n", response)

	if strings.ToLower(response) == "+pong" {
		replConfig := fmt.Sprintf("*3\r\n$8\r\nREPLCONF\r\n$14\r\nlistening-port\r\n$4\r\n%s\r\n", replicaPort)

		m.Write([]byte(replConfig))
		m.Write([]byte("*3\r\n$8\r\nREPLCONF\r\n$4\r\ncapa\r\n$6\r\npsync2\r\n"))
		m.Write([]byte("*3\r\n$5\r\nPSYNC\r\n$1\r\n?\r\n$2\r\n-1\r\n"))
	} else {
		fmt.Println("Received unexpected response")
	}
}
