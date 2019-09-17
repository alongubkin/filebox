package client

import (
	"encoding/gob"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alongubkin/filebox/pkg/protocol"
	log "github.com/sirupsen/logrus"
)

// FileboxClient is responsible for managing the client side of the Filebox protocol.
// In order to create a new FileboxClient, use the Connect method.
type FileboxClient struct {
	connection    net.Conn
	nextMessageID uint32
	encoder       *gob.Encoder
	decoder       *gob.Decoder
	channels      sync.Map
}

func Connect(address string, exit chan struct{}) (*FileboxClient, error) {
	connection, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	client := &FileboxClient{
		connection, 0,
		gob.NewEncoder(connection), gob.NewDecoder(connection),
		sync.Map{},
	}

	go client.handleMessages(exit)
	return client, nil
}

func (client *FileboxClient) SendReceive(data interface{}) (interface{}, bool) {
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
		return nil, false
	}

	// Wait for response
	select {
	case response := <-responseChannel:
		return response.Data, response.Success

	case <-time.After(3 * time.Second):
		return nil, false
	}
}

func (client *FileboxClient) handleMessages(exit chan struct{}) {
	for {
		message := &protocol.Message{}
		if err := client.decoder.Decode(message); err != nil {
			log.WithError(err).Error("decoder.Decode() failed")
			close(exit)
			return
		}

		// TODO: Validate IsResponse

		if channel, ok := client.channels.Load(message.MessageID); ok {
			channel.(chan *protocol.Message) <- message
		} else {
			log.WithField("MessageID", message.MessageID).Warn("Didn't find response channel for message.")
		}
	}
}
