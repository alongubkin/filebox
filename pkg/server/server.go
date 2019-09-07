package server

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"

	"github.com/alongubkin/filebox/pkg/protocol"
)

func handleMessage(messageHandler *FileboxMessageHandler, encoder *gob.Encoder, message *protocol.Message) {
	// TODO: Validate !IsResponse
	// TODO: Validate ID is correct
	response := &protocol.Message{
		MessageID:  message.MessageID,
		IsResponse: true,
	}

	switch request := message.Data.(type) {
	case protocol.ReadDirectoryRequestMessage:
		if data := messageHandler.ReadDirectory(request); data != nil {
			response.Data = data
		}

	case protocol.GetFileAttributesRequestMessage:
		if data := messageHandler.GetFileAttributes(request); data != nil {
			response.Data = data
		}
	}

	if err := encoder.Encode(response); err != nil {
		log.Fatal(err)
	}
}

func handleConnection(messageHandler *FileboxMessageHandler, connection net.Conn) {
	encoder := gob.NewEncoder(connection)
	decoder := gob.NewDecoder(connection)

	for {
		message := &protocol.Message{}

		err := decoder.Decode(message)
		if err == nil {
			go handleMessage(messageHandler, encoder, message)
		}
	}
}

func RunServer(basePath string, port uint16) {
	listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Println(err)
		return
	}

	defer listener.Close()

	messageHandler := &FileboxMessageHandler{BasePath: basePath}

	for {
		connection, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}

		go handleConnection(messageHandler, connection)
	}
}
