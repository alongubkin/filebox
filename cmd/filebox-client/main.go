package main

import (
	"github.com/alongubkin/filebox/pkg/client"
	"github.com/alongubkin/filebox/pkg/protocol"
	"github.com/billziss-gh/cgofuse/fuse"
	log "github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	verbose    = kingpin.Flag("verbose", "Verbose mode.").Short('v').Bool()
	address    = kingpin.Flag("address", "Remote address of the Filebox server").Required().Short('r').String()
	mountpoint = kingpin.Flag("mountpoint", "Path to mount the Filebox directory.").Required().Short('m').String()
)

func main() {
	kingpin.Parse()

	if *verbose {
		log.SetLevel(log.TraceLevel)
	}

	protocol.Init()

	c, err := client.Connect(*address)
	if err != nil {
		log.WithError(err).Fatal("Can't connect to Filebox server")
		return
	}

	log.Info("Connected.")

	fs := &client.FileboxFileSystem{Client: c}
	host := fuse.NewFileSystemHost(fs)
	host.Mount(*mountpoint, []string{"-o", "direct_io"})
}
