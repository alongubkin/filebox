package server

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"sync/atomic"

	"github.com/alongubkin/filebox/pkg/protocol"
)

type FileboxMessageHandler struct {
	BasePath       string
	fileHandles    sync.Map
	nextFileHandle uint64
}

func (handler *FileboxMessageHandler) OpenFile(request protocol.OpenFileRequestMessage) *protocol.OpenFileResponseMessage {
	file, err := os.OpenFile(path.Join(handler.BasePath, request.Path), request.Flags, 0755)
	if err != nil {
		return nil
	}

	fileHandle := atomic.AddUint64(&handler.nextFileHandle, 1)
	handler.fileHandles.Store(fileHandle, file)

	return &protocol.OpenFileResponseMessage{
		FileHandle: fileHandle,
	}
}

func (handler *FileboxMessageHandler) ReadFile(request protocol.ReadFileRequestMessage) *protocol.ReadFileResponseMessage {
	file, ok := handler.fileHandles.Load(request.FileHandle)
	if !ok {
		return nil
	}

	buff := make([]byte, request.Size)

	bytesRead, err := file.(*os.File).ReadAt(buff, request.Offset)
	if err != nil && err != io.EOF {
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
		return nil
	}

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
		file, ok := handler.fileHandles.Load(request.FileHandle)
		if !ok {
			return nil
		}

		fileInfo, err = file.(*os.File).Stat()
		if err != nil {
			return nil
		}
	} else {
		fileInfo, err = os.Stat(path.Join(handler.BasePath, request.Path))
		if err != nil {
			return nil
		}
	}

	return &protocol.GetFileAttributesResponseMessage{
		FileInfo: convertFileInfo(fileInfo),
	}
}

func (handler *FileboxMessageHandler) CloseFile(request protocol.CloseFileRequestMessage) *protocol.CloseFileResponseMessage {
	file, ok := handler.fileHandles.Load(request.FileHandle)
	if !ok {
		return nil
	}

	fmt.Printf("Closing %d\n", request.FileHandle)
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
