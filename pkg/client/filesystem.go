package client

import (
	"github.com/alongubkin/filebox/pkg/protocol"
	"github.com/billziss-gh/cgofuse/fuse"
	log "github.com/sirupsen/logrus"
)

type Hellofs struct {
	fuse.FileSystemBase
	Client *FileboxClient
}

func (self *Hellofs) Open(path string, flags int) (errc int, fh uint64) {
	file, err := self.Client.OpenFile(path, flags)
	if err != nil {
		log.WithField("path", path).WithError(err).Error("OpenFile failed")
		return -fuse.ENOENT, ^uint64(0)
	}

	log.WithFields(log.Fields{
		"fh":    file,
		"flags": flags,
	}).Tracef("Opened file %s", path)

	return 0, file
}

func (self *Hellofs) Getattr(path string, stat *fuse.Stat_t, fh uint64) (errc int) {
	log.Tracef("Get file attributes %s", path)

	file, err := self.Client.GetFileAttributes(path, fh)
	if err != nil {
		log.WithField("path", path).WithError(err).Error("GetFileAttributes failed")
		return -fuse.ENOENT
	}

	*stat = *convertFileInfo(file)
	return 0
}

func (self *Hellofs) Read(path string, buff []byte, ofst int64, fh uint64) (n int) {
	log.WithFields(log.Fields{
		"offset": ofst,
		"size":   len(buff),
	}).Tracef("Reading file %s", path)

	n, err := self.Client.ReadFile(fh, buff, ofst)
	if err != nil {
		log.WithField("path", path).WithError(err).Error("ReadFile failed")
		return n
	}

	return 0
}

func (self *Hellofs) Readdir(path string,
	fill func(name string, stat *fuse.Stat_t, ofst int64) bool,
	ofst int64,
	fh uint64) (errc int) {

	log.WithField("offset", ofst).Tracef("Reading directory %s", path)

	fill(".", nil, 0)
	fill("..", nil, 0)

	files, err := self.Client.ReadDirectory(path)
	if err != nil {
		log.WithField("path", path).WithError(err).Error("ReadDirectory failed")
		return -fuse.ENOENT
	}

	for _, file := range files {
		if !fill(file.Name, convertFileInfo(&file), 0) {
			break
		}
	}

	return 0
}

func (self *Hellofs) Release(path string, fh uint64) int {
	log.WithField("fh", fh).Tracef("Closing file %s", path)

	if err := self.Client.CloseFile(fh); err != nil {
		log.WithField("path", path).WithError(err).Error("CloseFile failed")
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
