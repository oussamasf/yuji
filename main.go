package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/oussamasf/yuji/utils"
)

func main() {
	port := flag.String("p", "8080", "port")
	flag.Parse()

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

		go utils.HandleConnection(conn, redisMap)
	}
}
