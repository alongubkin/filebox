package server

import (
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"reflect"

	"github.com/alongubkin/filebox/pkg/protocol"
	log "github.com/sirupsen/logrus"
)

func handleMessage(messageHandler *FileboxMessageHandler, encoder *gob.Encoder, message *protocol.Message) {
	if message.IsResponse {
		log.Warn("Got a message from a client with IsResponse flag turned on. Ignoring")
	}

	response := &protocol.Message{
		MessageID:  message.MessageID,
		IsResponse: true,
	}

	var err error
	var data interface{}

	switch request := message.Data.(type) {
	case protocol.OpenFileRequest:
		data, err = messageHandler.OpenFile(request)

	case protocol.ReadFileRequest:
		data, err = messageHandler.ReadFile(request)

	case protocol.ReadDirectoryRequest:
		data, err = messageHandler.ReadDirectory(request)

	case protocol.GetFileAttributesRequest:
		data, err = messageHandler.GetFileAttributes(request)

	case protocol.CloseFileRequest:
		err = messageHandler.CloseFile(request)

	case protocol.CreateDirectoryRequest:
		err = messageHandler.CreateDirectory(request)

	case protocol.CreateFileRequest:
		err = messageHandler.CreateFile(request)

	case protocol.RenameRequest:
		err = messageHandler.Rename(request)

	case protocol.DeleteDirectoryRequest:
		err = messageHandler.DeleteDirectory(request)

	case protocol.TruncateRequest:
		err = messageHandler.Truncate(request)

	case protocol.DeleteFileRequest:
		err = messageHandler.DeleteFile(request)

	case protocol.WriteFileRequest:
		data, err = messageHandler.WriteFile(request)
	}

	// data == nil won't work here because in Go, nil.(interface{}) != nil.(MyCommandResponse)
	if val := reflect.ValueOf(data); !val.IsValid() || val.IsNil() {
		response.Data = &protocol.EmptyResponse{}
	} else {
		response.Data = data
	}

	response.Success = (err == nil)

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
