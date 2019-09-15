package main

import (
	"os"

	"github.com/alongubkin/filebox/pkg/client"
	"github.com/alongubkin/filebox/pkg/protocol"
	"github.com/billziss-gh/cgofuse/fuse"
	log "github.com/sirupsen/logrus"
)

func main() {
	// TODO: Arg: Mountpoint
	log.SetLevel(log.TraceLevel)
	protocol.Init()

	c, err := client.Connect("127.0.0.1:8763")
	if err != nil {
		log.WithError(err).Fatal("Can't connect to Filebox server")
		return
	}

	log.Info("Connected.")

	hellofs := &client.Hellofs{Client: c}
	host := fuse.NewFileSystemHost(hellofs)
	host.Mount(os.Args[1], []string{"-o", "direct_io"})
}
