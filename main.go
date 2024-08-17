package main

import (
	"flag"
	"fmt"
	"net"
	"strings"

	"github.com/oussamasf/yuji/utils"
)

// ? Config holds the server configuration.

func main() {
	var r string
	var RSlice []string

	//? Config object to hold all the configuration variables
	config := &utils.Config{
		RedisMap: make(map[string]string),
		IsSlave:  false,
	}

	//? Parse command-line flags
	flag.StringVar(&config.Port, "p", "8080", "port")
	flag.StringVar(&config.ReplicaType, "replicaof", "", "replica of")
	flag.StringVar(&config.Dir, "dir", "", "Directory to store RDB file")
	flag.StringVar(&config.DBFileName, "dbfilename", "dump.rdb", "RDB file name")

	flag.Parse()

	flag.Parse()

	if config.ReplicaType != "" {
		r = strings.TrimSpace(config.ReplicaType)
		RSlice = strings.Split(r, ":")

		if len(RSlice) != 2 {
			fmt.Println("INVALID_REPLICA_ARGUMENT")
			return
		}

		if RSlice[1] == config.Port {
			fmt.Println("PORT_OF_REPLICA_SHOULD_BE_DIFFERENT_FROM_MASTER")
			return
		}

		config.IsSlave = true
		go utils.HandleReplicaConnection(RSlice[0], RSlice[1], config.Port, config.RedisMap)
	}

	listener, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server is listening on " + ":" + config.Port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go utils.HandleConnection(conn, config)
	}
}
