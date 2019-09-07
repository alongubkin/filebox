package client

import (
	"fmt"

	"github.com/alongubkin/filebox/pkg/protocol"
	"github.com/billziss-gh/cgofuse/fuse"
)

const (
	filename = "hello"
	contents = "hello, world\n"
)

type Hellofs struct {
	fuse.FileSystemBase
	Client *FileboxClient
}

func (self *Hellofs) Open(path string, flags int) (errc int, fh uint64) {
	fmt.Printf("Open %s", path)
	switch path {
	case "/" + filename:
		return 0, 0
	default:
		return -fuse.ENOENT, ^uint64(0)
	}
}

func (self *Hellofs) Getattr(path string, stat *fuse.Stat_t, fh uint64) (errc int) {
	fmt.Printf("Getattr %s\n", path)

	file, err := self.Client.GetFileAttributes(path)
	if err != nil {
		return -fuse.ENOENT
	}

	*stat = *convertFileInfo(file)

	// Mode = fuse.S_IFDIR | 0555
	// stat.Size = file.Size

	// *stat = *convertFileInfo(file)
	return 0
	// stat = convertFileInfo(file)
}

func (self *Hellofs) Read(path string, buff []byte, ofst int64, fh uint64) (n int) {
	endofst := ofst + int64(len(buff))
	if endofst > int64(len(contents)) {
		endofst = int64(len(contents))
	}
	if endofst < ofst {
		return 0
	}
	n = copy(buff, contents[ofst:endofst])
	return
}

func (self *Hellofs) Readdir(path string,
	fill func(name string, stat *fuse.Stat_t, ofst int64) bool,
	ofst int64,
	fh uint64) (errc int) {

	fill(".", nil, 0)
	fill("..", nil, 0)

	files, err := self.Client.ReadDirectory(path)
	if err != nil {
		return -fuse.ENOENT
	}

	for _, file := range files {
		if !fill(file.Name, convertFileInfo(&file), 0) {
			break
		}
	}

	return 0
}

func convertFileInfo(file *protocol.FileInfo) *fuse.Stat_t {
	var mode uint32 = 0555

	if file.Mode.IsDir() {
		mode |= fuse.S_IFDIR
	}

	if file.Mode.IsRegular() {
		mode |= fuse.S_IFREG
	}

	return &fuse.Stat_t{
		Mode: mode,
		Size: file.Size,
		Mtim: fuse.NewTimespec(file.ModTime),
	}
}
