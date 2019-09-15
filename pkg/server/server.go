package server

import (
	"encoding/gob"
	"fmt"
	"io"
	"net"

	"github.com/alongubkin/filebox/pkg/protocol"
	log "github.com/sirupsen/logrus"
)

func handleMessage(messageHandler *FileboxMessageHandler, encoder *gob.Encoder, message *protocol.Message) {
	// TODO: Validate !IsResponse
	// TODO: Validate ID is correct
	response := &protocol.Message{
		MessageID:  message.MessageID,
		IsResponse: true,
	}

	switch request := message.Data.(type) {
	case protocol.OpenFileRequestMessage:
		if data := messageHandler.OpenFile(request); data != nil {
			response.Data = data
		}

	case protocol.ReadFileRequestMessage:
		if data := messageHandler.ReadFile(request); data != nil {
			response.Data = data
		}

	case protocol.ReadDirectoryRequestMessage:
		if data := messageHandler.ReadDirectory(request); data != nil {
			response.Data = data
		}

	case protocol.GetFileAttributesRequestMessage:
		if data := messageHandler.GetFileAttributes(request); data != nil {
			response.Data = data
		}

	case protocol.CloseFileRequestMessage:
		if data := messageHandler.CloseFile(request); data != nil {
			response.Data = data
		}
	}

	if err := encoder.Encode(response); err != nil {
		log.WithError(err).Error("encoder.Encode failed")
	}
}

func handleConnection(messageHandler *FileboxMessageHandler, connection net.Conn) {
	encoder := gob.NewEncoder(connection)
	decoder := gob.NewDecoder(connection)

	log.WithField("address", connection.RemoteAddr()).Info("Handling new connection")

	for {
		message := &protocol.Message{}

		err := decoder.Decode(message)
		if err == io.EOF {
			log.WithField("address", connection.RemoteAddr()).Info("Disconnected.")
			return
		} else if err != nil {
			log.WithError(err).Error("decoder.Decode failed")
			return
		}

		go handleMessage(messageHandler, encoder, message)
	}
}

func RunServer(basePath string, port uint16) {
	listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", port))
	if err != nil {
		log.WithError(err).WithField("port", port).Error("net.Listen() failed")
		return
	}

	defer listener.Close()

	messageHandler := &FileboxMessageHandler{BasePath: basePath}

	log.WithField("port", port).Info("Started.")

	for {
		connection, err := listener.Accept()
		if err != nil {
			log.WithError(err).Error("listener.Accept() failed.")
			return
		}

		go handleConnection(messageHandler, connection)
	}
}
