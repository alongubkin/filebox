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
	response, ok := fs.Client.SendReceive(protocol.OpenFileRequest{
		Path:  path,
		Flags: flags,
	})

	if !ok {
		log.WithField("path", path).Error("OpenFile failed")
		return -fuse.ENOENT, ^uint64(0)
	}

	log.WithFields(log.Fields{
		"fh":    response.(protocol.OpenFileResponse).FileHandle,
		"flags": flags,
	}).Tracef("Opened file %s", path)

	return 0, response.(protocol.OpenFileResponse).FileHandle
}

// Getattr gets file attributes.
func (fs *FileboxFileSystem) Getattr(path string, stat *fuse.Stat_t, fh uint64) (errc int) {
	log.Tracef("Get file attributes %s", path)

	response, ok := fs.Client.SendReceive(protocol.GetFileAttributesRequest{
		Path:       path,
		FileHandle: fh,
	})

	log.Tracef("Get file attributes2 %s", path)

	if !ok {
		log.WithField("path", path).Error("GetFileAttributes failed")
		return -fuse.ENOENT
	}

	log.Tracef("Get file attributes3 %s", path)

	fileInfo := response.(protocol.GetFileAttributesResponse).FileInfo
	log.Tracef("Get file attributes4 %s", path)

	*stat = *convertFileInfo(&fileInfo)
	log.Tracef("Get file attributes5 %s", path)

	return 0
}

// Read reads data from a file.
func (fs *FileboxFileSystem) Read(path string, buff []byte, ofst int64, fh uint64) (n int) {
	log.WithFields(log.Fields{
		"offset": ofst,
		"size":   len(buff),
	}).Tracef("Reading file %s", path)

	response, ok := fs.Client.SendReceive(protocol.ReadFileRequest{
		FileHandle: fh,
		Offset:     ofst,
		Size:       len(buff),
	})

	if !ok {
		log.WithField("path", path).Error("ReadFile failed")
		return 0
	}

	copy(buff, response.(protocol.ReadFileResponse).Data)
	return response.(protocol.ReadFileResponse).BytesRead
}

// Readdir reads a directory.
func (fs *FileboxFileSystem) Readdir(path string,
	fill func(name string, stat *fuse.Stat_t, ofst int64) bool,
	ofst int64,
	fh uint64) (errc int) {

	log.WithField("offset", ofst).Tracef("Reading directory %s", path)

	fill(".", nil, 0)
	fill("..", nil, 0)

	response, ok := fs.Client.SendReceive(protocol.ReadDirectoryRequest{
		Path: path,
	})

	if !ok {
		log.WithField("path", path).Error("ReadDirectory failed")
		return -fuse.ENOENT
	}

	for _, file := range response.(protocol.ReadDirectoryResponse).Files {
		if !fill(file.Name, convertFileInfo(&file), 0) {
			break
		}
	}

	return 0
}

// Release closes an open file.
func (fs *FileboxFileSystem) Release(path string, fh uint64) int {
	log.WithField("fh", fh).Tracef("Closing file %s", path)

	if _, ok := fs.Client.SendReceive(protocol.CloseFileRequest{fh}); !ok {
		log.WithField("path", path).Error("CloseFile failed")
	}

	return 0
}

// Mkdir creates a directory.
func (fs *FileboxFileSystem) Mkdir(path string, mode uint32) int {
	log.Tracef("Creating directory %s", path)

	if _, ok := fs.Client.SendReceive(protocol.CreateDirectoryRequest{path, mode}); !ok {
		log.WithField("path", path).Error("CreateDirectory failed")
		return -fuse.EIO
	}

	return 0
}

// Mknod creates a file.
func (fs *FileboxFileSystem) Mknod(path string, mode uint32, dev uint64) int {
	log.Tracef("Creating file %s", path)

	if (mode & fuse.S_IFREG) == 0 {
		log.WithFields(log.Fields{
			"path": path,
			"mode": mode,
			"dev":  dev,
		}).Errorf("Invalid file mode. ")
		return fuse.EINVAL
	}

	if _, ok := fs.Client.SendReceive(protocol.CreateFileRequest{path}); !ok {
		log.WithField("path", path).Error("CreateFile failed")
		return -fuse.EIO
	}

	return 0
}

// Rename renames a file.
func (fs *FileboxFileSystem) Rename(oldpath string, newpath string) int {
	log.Tracef("Renaming %s to %s", oldpath, newpath)

	_, ok := fs.Client.SendReceive(protocol.RenameRequest{
		OldPath: oldpath,
		NewPath: newpath,
	})

	if !ok {
		log.WithFields(log.Fields{
			"oldpath": oldpath,
			"newpath": newpath,
		}).Error("Rename failed")
	}

	return 0
}

// Rmdir removes a directory.
func (fs *FileboxFileSystem) Rmdir(path string) int {
	log.Tracef("Deleting directory %s", path)

	if _, ok := fs.Client.SendReceive(protocol.DeleteDirectoryRequest{path}); !ok {
		log.WithField("path", path).Error("DeleteDirectory failed")
		return -fuse.EIO
	}

	return 0
}

// Truncate changes the size of a file.
func (fs *FileboxFileSystem) Truncate(path string, size int64, fh uint64) int {
	log.Tracef("Truncating %s", path)

	_, ok := fs.Client.SendReceive(protocol.TruncateRequest{
		Path:       path,
		Size:       size,
		FileHandle: fh,
	})

	if !ok {
		log.WithFields(log.Fields{
			"path": path,
			"size": size,
			"fh":   fh,
		}).Error("Truncate failed")
		return -fuse.EIO
	}

	return 0
}

// Unlink removes a file.
func (fs *FileboxFileSystem) Unlink(path string) int {
	log.Tracef("Deleting file %s", path)

	if _, ok := fs.Client.SendReceive(protocol.DeleteFileRequest{path}); !ok {
		log.WithField("path", path).Error("Truncate failed")
		return -fuse.EIO
	}

	return 0
}

// Write writes data to a file.
func (fs *FileboxFileSystem) Write(path string, buff []byte, ofst int64, fh uint64) int {
	log.WithFields(log.Fields{
		"offset": ofst,
		"size":   len(buff),
	}).Tracef("Writing file %s", path)

	response, ok := fs.Client.SendReceive(protocol.WriteFileRequest{
		FileHandle: fh,
		Data:       buff,
		Offset:     ofst,
	})

	if !ok {
		log.WithField("path", path).Error("WriteFile failed")
		return -fuse.EIO
	}

	return response.(protocol.WriteFileResponse).BytesWritten
}

func convertFileInfo(file *protocol.FileInfo) *fuse.Stat_t {
	var mode uint32 = 0777

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
