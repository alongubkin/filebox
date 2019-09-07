package main

import (
	"github.com/alongubkin/filebox/pkg/protocol"
	"github.com/alongubkin/filebox/pkg/server"
)

func main() {
	// TODO: Arg: directory
	protocol.Init()
	server.RunServer("/tmp/x", 8763)
}
