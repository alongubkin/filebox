package server

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/alongubkin/filebox/pkg/protocol"
)

func handleConnection(client net.Conn) {
	fmt.Printf("Serving %s\n", client.RemoteAddr().String())
	for {
		netData, err := bufio.NewReader(client).ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}

		temp := strings.TrimSpace(string(netData))
		if temp == "STOP" {
			break
		}

		result := strconv.Itoa(1234) + "\n"
		client.Write([]byte(string(result)))
	}
	client.Close()
}

func Run() {
	listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", protocol.FileboxPort))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer listener.Close()

	for {
		client, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}

		go handleConnection(client)
	}
}
