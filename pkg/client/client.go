package client

import (
	"encoding/gob"
	"net"
	"sync"
	"sync/atomic"

	"github.com/alongubkin/filebox/pkg/protocol"
	log "github.com/sirupsen/logrus"
)

type FileboxClient struct {
	connection    net.Conn
	nextMessageID uint32
	encoder       *gob.Encoder
	decoder       *gob.Decoder
	channels      sync.Map
}

func Connect(address string) (*FileboxClient, error) {
	connection, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	client := &FileboxClient{
		connection, 0,
		gob.NewEncoder(connection), gob.NewDecoder(connection),
		sync.Map{},
	}

	go client.handleMessages()
	return client, nil
}

func (client *FileboxClient) OpenFile(path string, flags int) (uint64, error) {
	data, err := client.sendAndReceiveMessage(protocol.OpenFileRequestMessage{path, flags})
	if err != nil {
		return 0, err
	}

	response := data.(protocol.OpenFileResponseMessage)
	return response.FileHandle, nil
}

func (client *FileboxClient) ReadFile(fileHandle uint64, buff []byte, offset int64) (int, error) {
	data, err := client.sendAndReceiveMessage(protocol.ReadFileRequestMessage{fileHandle, offset, len(buff)})
	if err != nil {
		return 0, err
	}

	response := data.(protocol.ReadFileResponseMessage)
	copy(buff, response.Data)

	return response.BytesRead, err
}

func (client *FileboxClient) ReadDirectory(path string) ([]protocol.FileInfo, error) {
	data, err := client.sendAndReceiveMessage(protocol.ReadDirectoryRequestMessage{path})
	if err != nil {
		return nil, err
	}

	response := data.(protocol.ReadDirectoryResponseMessage)
	return response.Files, nil
}

func (client *FileboxClient) GetFileAttributes(path string, fileHandle uint64) (*protocol.FileInfo, error) {
	data, err := client.sendAndReceiveMessage(protocol.GetFileAttributesRequestMessage{path, fileHandle})
	if err != nil {
		return nil, err
	}

	response := data.(protocol.GetFileAttributesResponseMessage)
	return &response.FileInfo, nil
}

func (client *FileboxClient) CloseFile(fileHandle uint64) error {
	_, err := client.sendAndReceiveMessage(protocol.CloseFileRequestMessage{fileHandle})
	if err != nil {
		return err
	}

	return nil
}

func (client *FileboxClient) sendAndReceiveMessage(data interface{}) (interface{}, error) {
	// Calculate message ID atomically
	messageID := atomic.AddUint32(&client.nextMessageID, 1)

	// Create the response channel
	responseChannel := make(chan *protocol.Message)
	client.channels.Store(messageID, responseChannel)
	defer client.channels.Delete(messageID)

	// Send message
	message := &protocol.Message{
		MessageID:  messageID,
		IsResponse: false,
		Data:       data,
	}
	if err := client.encoder.Encode(message); err != nil {
		return nil, err
	}

	// Wait for response
	response := <-responseChannel
	return response.Data, nil
}

func (client *FileboxClient) handleMessages() {

	for {
		message := &protocol.Message{}
		if err := client.decoder.Decode(message); err != nil {
			log.WithError(err).Fatalf("decoder.Decode() failed")
			return
		}

		if channel, ok := client.channels.Load(message.MessageID); ok {
			channel.(chan *protocol.Message) <- message
		} else {
			log.WithField("MessageID", message.MessageID).Warn("Didn't find response channel for message.")
		}
	}
}
