package client

import (
	"github.com/alongubkin/filebox/pkg/protocol"
	"github.com/billziss-gh/cgofuse/fuse"
	log "github.com/sirupsen/logrus"
)

type FileboxFileSystem struct {
	fuse.FileSystemBase
	Client *FileboxClient
}

// Open opens a file.
// The flags are a combination of the fuse.O_* constants.
func (fs *FileboxFileSystem) Open(path string, flags int) (errc int, fh uint64) {
	file, err := fs.Client.OpenFile(path, flags)
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

// Getattr gets file attributes.
func (fs *FileboxFileSystem) Getattr(path string, stat *fuse.Stat_t, fh uint64) (errc int) {
	log.Tracef("Get file attributes %s", path)

	file, err := fs.Client.GetFileAttributes(path, fh)
	if err != nil {
		log.WithField("path", path).WithError(err).Error("GetFileAttributes failed")
		return -fuse.ENOENT
	}

	*stat = *convertFileInfo(file)
	return 0
}

// Read reads data from a file.
func (fs *FileboxFileSystem) Read(path string, buff []byte, ofst int64, fh uint64) (n int) {
	log.WithFields(log.Fields{
		"offset": ofst,
		"size":   len(buff),
	}).Tracef("Reading file %s", path)

	n, err := fs.Client.ReadFile(fh, buff, ofst)
	if err != nil {
		log.WithField("path", path).WithError(err).Error("ReadFile failed")
		return n
	}

	return 0
}

// Readdir reads a directory.
func (fs *FileboxFileSystem) Readdir(path string,
	fill func(name string, stat *fuse.Stat_t, ofst int64) bool,
	ofst int64,
	fh uint64) (errc int) {

	log.WithField("offset", ofst).Tracef("Reading directory %s", path)

	fill(".", nil, 0)
	fill("..", nil, 0)

	files, err := fs.Client.ReadDirectory(path)
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

// Release closes an open file.
func (fs *FileboxFileSystem) Release(path string, fh uint64) int {
	log.WithField("fh", fh).Tracef("Closing file %s", path)

	if err := fs.Client.CloseFile(fh); err != nil {
		log.WithField("path", path).WithError(err).Error("CloseFile failed")
	}

	return 0
}

// func (*FileSystemBase) Mkdir(path string, mode uint32) int
// func (*FileSystemBase) Mknod(path string, mode uint32, dev uint64) int
// func (*FileSystemBase) Rename(oldpath string, newpath string) int
// func (*FileSystemBase) Rmdir(path string) int
// func (*FileSystemBase) Truncate(path string, size int64, fh uint64) int
// func (*FileSystemBase) Unlink(path string) int
// func (*FileSystemBase) Write(path string, buff []byte, ofst int64, fh uint64) int

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
