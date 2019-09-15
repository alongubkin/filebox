package main

import (
	"github.com/alongubkin/filebox/pkg/protocol"
	"github.com/alongubkin/filebox/pkg/server"
	log "github.com/sirupsen/logrus"
)

func main() {
	// TODO: Arg: directory
	log.SetLevel(log.TraceLevel)
	protocol.Init()
	server.RunServer("/tmp/x", 8763)
}
