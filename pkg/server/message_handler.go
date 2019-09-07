package server

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/alongubkin/filebox/pkg/protocol"
)

type FileboxMessageHandler struct {
	BasePath string
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
	file, err := os.Stat(path.Join(handler.BasePath, request.Path))
	if err != nil {
		return nil
	}

	response := &protocol.GetFileAttributesResponseMessage{
		FileInfo: convertFileInfo(file),
	}

	return response
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
