package server

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"sync/atomic"

	"github.com/alongubkin/filebox/pkg/protocol"
	log "github.com/sirupsen/logrus"
)

type FileboxMessageHandler struct {
	BasePath       string
	fileHandles    sync.Map
	nextFileHandle uint64
}

func (handler *FileboxMessageHandler) OpenFile(request protocol.OpenFileRequestMessage) *protocol.OpenFileResponseMessage {
	file, err := os.OpenFile(path.Join(handler.BasePath, request.Path), request.Flags, 0755)
	if err != nil {
		log.WithField("path", request.Path).WithError(err).Error("OpenFile failed")
		return nil
	}

	fileHandle := atomic.AddUint64(&handler.nextFileHandle, 1)
	handler.fileHandles.Store(fileHandle, file)

	log.WithFields(log.Fields{
		"fh":    fileHandle,
		"flags": request.Flags,
	}).Tracef("Opened file %s", request.Path)

	return &protocol.OpenFileResponseMessage{
		FileHandle: fileHandle,
	}
}

func (handler *FileboxMessageHandler) ReadFile(request protocol.ReadFileRequestMessage) *protocol.ReadFileResponseMessage {
	file, ok := handler.fileHandles.Load(request.FileHandle)
	if !ok {
		log.WithField("fh", request.FileHandle).Error("Invalid file handle in ReadFile request")
		return nil
	}

	log.WithFields(log.Fields{
		"fh":     request.FileHandle,
		"offset": request.Offset,
		"size":   request.Size,
	}).Tracef("Reading file %s", file.(*os.File).Name())

	buff := make([]byte, request.Size)

	bytesRead, err := file.(*os.File).ReadAt(buff, request.Offset)
	if err != nil && err != io.EOF {
		log.WithFields(log.Fields{
			"fh":     request.FileHandle,
			"offset": request.Offset,
			"size":   request.Size,
		}).WithError(err).Error("file.ReadAt failed")
		return nil
	}

	return &protocol.ReadFileResponseMessage{
		Data:      buff,
		BytesRead: bytesRead,
	}
}

func (handler *FileboxMessageHandler) ReadDirectory(request protocol.ReadDirectoryRequestMessage) *protocol.ReadDirectoryResponseMessage {
	files, err := ioutil.ReadDir(path.Join(handler.BasePath, request.Path))
	if err != nil {
		log.WithField("path", request.Path).WithError(err).Error("ReadDir failed")
		return nil
	}

	log.Tracef("Reading directory %s", request.Path)

	response := &protocol.ReadDirectoryResponseMessage{}
	for _, file := range files {
		response.Files = append(response.Files, convertFileInfo(file))
	}

	return response
}

func (handler *FileboxMessageHandler) GetFileAttributes(request protocol.GetFileAttributesRequestMessage) *protocol.GetFileAttributesResponseMessage {
	var fileInfo os.FileInfo
	var err error

	if request.FileHandle <= handler.nextFileHandle {
		log.WithField("fh", request.FileHandle).Tracef("Get file attributes %s", request.Path)

		file, ok := handler.fileHandles.Load(request.FileHandle)
		if !ok {
			log.WithField("fh", request.FileHandle).Error("Invalid file handle in GetFileAttributes request")
			return nil
		}

		fileInfo, err = file.(*os.File).Stat()
		if err != nil {
			log.WithFields(log.Fields{
				"path": request.Path,
			}).WithError(err).Error("file.Stat() failed")
			return nil
		}
	} else {
		fileInfo, err = os.Stat(path.Join(handler.BasePath, request.Path))
		if err != nil {
			log.WithFields(log.Fields{
				"path": request.Path,
			}).WithError(err).Warn("os.Stat() failed")
			return nil
		}
	}

	return &protocol.GetFileAttributesResponseMessage{
		FileInfo: convertFileInfo(fileInfo),
	}
}

func (handler *FileboxMessageHandler) CloseFile(request protocol.CloseFileRequestMessage) *protocol.CloseFileResponseMessage {
	log.WithFields(log.Fields{
		"fh": request.FileHandle,
	}).Tracef("Close file")

	file, ok := handler.fileHandles.Load(request.FileHandle)
	if !ok {
		log.WithField("fh", request.FileHandle).Error("Invalid file handle in CloseFile request")
		return nil
	}

	file.(*os.File).Close()
	handler.fileHandles.Delete(request.FileHandle)

	return &protocol.CloseFileResponseMessage{}
}

func convertFileInfo(file os.FileInfo) protocol.FileInfo {
	return protocol.FileInfo{
		Name:    file.Name(),
		Size:    file.Size(),
		Mode:    file.Mode(),
		ModTime: file.ModTime(),
		IsDir:   file.IsDir(),
	}
}
