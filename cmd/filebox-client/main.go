package main

import (
	"fmt"
	"os"

	"github.com/alongubkin/filebox/pkg/client"
	"github.com/alongubkin/filebox/pkg/protocol"
	"github.com/billziss-gh/cgofuse/fuse"
)

func main() {
	// TODO: Arg: Mountpoint
	protocol.Init()

	c, err := client.Connect("127.0.0.1:8763")
	if err != nil {
		fmt.Printf("%d", err)
		return
	}

	hellofs := &client.Hellofs{Client: c}
	host := fuse.NewFileSystemHost(hellofs)
	host.Mount(os.Args[1], []string{"-o", "direct_io"})
}
