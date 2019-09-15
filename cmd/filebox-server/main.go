package main

import (
	"github.com/alongubkin/filebox/pkg/protocol"
	"github.com/alongubkin/filebox/pkg/server"
	log "github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	verbose = kingpin.Flag("verbose", "Verbose mode.").Short('v').Bool()
	path    = kingpin.Flag("path", "Path to the shared directory.").Required().Short('d').String()
	port    = kingpin.Flag("port", "TCP Port to listen on.").Required().Short('p').Uint16()
)

func main() {
	kingpin.Parse()

	if *verbose {
		log.SetLevel(log.TraceLevel)
	}

	protocol.Init()
	server.RunServer(*path, *port)
}
