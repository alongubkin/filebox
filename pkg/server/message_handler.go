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
	BasePath string

	// FUTURE: Automatically close handles if their client is disconnected.
	fileHandles    sync.Map
	nextFileHandle uint64
}

func (handler *FileboxMessageHandler) OpenFile(request protocol.OpenFileRequest) (*protocol.OpenFileResponse, error) {
	file, err := os.OpenFile(path.Join(handler.BasePath, request.Path), request.Flags & ^os.O_EXCL, 00777)
	if err != nil {
		log.WithField("path", request.Path).WithError(err).Error("OpenFile failed")
		return nil, err
	}

	fileHandle := atomic.AddUint64(&handler.nextFileHandle, 1)
	handler.fileHandles.Store(fileHandle, file)

	log.WithFields(log.Fields{
		"fh":    fileHandle,
		"flags": request.Flags,
	}).Tracef("Opened file %s", request.Path)

	return &protocol.OpenFileResponse{
		FileHandle: fileHandle,
	}, nil
}

func (handler *FileboxMessageHandler) ReadFile(request protocol.ReadFileRequest) (*protocol.ReadFileResponse, error) {
	file, ok := handler.fileHandles.Load(request.FileHandle)
	if !ok {
		log.WithField("fh", request.FileHandle).Error("Invalid file handle in ReadFile request")
		return nil, os.ErrInvalid
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
		return nil, err
	}

	return &protocol.ReadFileResponse{
		Data:      buff,
		BytesRead: bytesRead,
	}, nil
}

func (handler *FileboxMessageHandler) ReadDirectory(request protocol.ReadDirectoryRequest) (*protocol.ReadDirectoryResponse, error) {
	files, err := ioutil.ReadDir(path.Join(handler.BasePath, request.Path))
	if err != nil {
		log.WithField("path", request.Path).WithError(err).Error("ReadDir failed")
		return nil, err
	}

	log.Tracef("Reading directory %s", request.Path)

	response := &protocol.ReadDirectoryResponse{}
	for _, file := range files {
		response.Files = append(response.Files, convertFileInfo(file))
	}

	return response, nil
}

func (handler *FileboxMessageHandler) GetFileAttributes(request protocol.GetFileAttributesRequest) (*protocol.GetFileAttributesResponse, error) {
	var fileInfo os.FileInfo
	var err error

	if request.FileHandle <= handler.nextFileHandle {
		log.WithField("fh", request.FileHandle).Tracef("Get file attributes %s", request.Path)

		file, ok := handler.fileHandles.Load(request.FileHandle)
		if !ok {
			log.WithField("fh", request.FileHandle).Error("Invalid file handle in GetFileAttributes request")
			return nil, os.ErrInvalid
		}

		fileInfo, err = file.(*os.File).Stat()
		if err != nil {
			log.WithField("path", file.(*os.File).Name()).WithError(err).Warn("file.Stat() failed")
			return nil, err
		}
	} else {
		fileInfo, err = os.Stat(path.Join(handler.BasePath, request.Path))
		if err != nil {
			log.WithField("path", request.Path).WithError(err).Warn("os.Stat() failed")
			return nil, err
		}
	}

	return &protocol.GetFileAttributesResponse{
		FileInfo: convertFileInfo(fileInfo),
	}, nil
}

func (handler *FileboxMessageHandler) CloseFile(request protocol.CloseFileRequest) error {
	log.WithField("fh", request.FileHandle).Tracef("Close file")

	file, ok := handler.fileHandles.Load(request.FileHandle)
	if !ok {
		log.WithField("fh", request.FileHandle).Error("Invalid file handle in CloseFile request")
		return os.ErrInvalid
	}

	file.(*os.File).Close()
	handler.fileHandles.Delete(request.FileHandle)

	return nil
}

func (handler *FileboxMessageHandler) CreateDirectory(request protocol.CreateDirectoryRequest) error {
	log.WithField("mode", request.Mode).Tracef("Creating directory %s", request.Path)

	err := os.Mkdir(path.Join(handler.BasePath, request.Path), os.FileMode(request.Mode))
	if err != nil {
		log.WithFields(log.Fields{
			"path": request.Path,
			"mode": request.Mode,
		}).WithError(err).Error("CreateDirectory failed")
		return err
	}

	return nil
}

func (handler *FileboxMessageHandler) CreateFile(request protocol.CreateFileRequest) error {
	log.Tracef("Creating file %s", request.Path)

	file, err := os.Create(path.Join(handler.BasePath, request.Path))
	if err != nil {
		log.WithField("path", request.Path).WithError(err).Error("CreateFile failed")
		return err
	}

	file.Close()
	return nil
}

func (handler *FileboxMessageHandler) Rename(request protocol.RenameRequest) error {
	log.Tracef("Renaming %s to %s", request.OldPath, request.NewPath)

	err := os.Rename(
		path.Join(handler.BasePath, request.OldPath),
		path.Join(handler.BasePath, request.NewPath),
	)

	if err != nil {
		log.WithFields(log.Fields{
			"old_path": request.OldPath,
			"new_path": request.NewPath,
		}).WithError(err).Error("Rename failed")
		return err
	}

	return nil
}

func (handler *FileboxMessageHandler) DeleteDirectory(request protocol.DeleteDirectoryRequest) error {
	log.Tracef("Deleting directory %s", request.Path)

	if err := os.RemoveAll(path.Join(handler.BasePath, request.Path)); err != nil {
		log.WithField("path", request.Path).WithError(err).Error("DeleteDirectory failed")
		return err
	}

	return nil
}

func (handler *FileboxMessageHandler) Truncate(request protocol.TruncateRequest) error {
	if request.FileHandle <= handler.nextFileHandle {
		log.WithFields(log.Fields{
			"fh":   request.FileHandle,
			"size": request.Size,
		}).Tracef("Truncating %s", request.Path)

		file, ok := handler.fileHandles.Load(request.FileHandle)
		if !ok {
			log.WithField("fh", request.FileHandle).Error("Invalid file handle in Truncate request")
			return os.ErrInvalid
		}

		if err := file.(*os.File).Truncate(request.Size); err != nil {
			log.WithFields(log.Fields{
				"path": file.(*os.File).Name(),
				"size": request.Size,
			}).WithError(err).Error("Truncate failed")
			return err
		}
	} else {
		err := os.Truncate(path.Join(handler.BasePath, request.Path), request.Size)
		if err != nil {
			log.WithFields(log.Fields{
				"path": request.Path,
				"size": request.Size,
			}).WithError(err).Error("Truncate failed")
			return err
		}
	}

	return nil
}

func (handler *FileboxMessageHandler) DeleteFile(request protocol.DeleteFileRequest) error {
	log.Tracef("Deleting file %s", request.Path)

	if err := os.Remove(path.Join(handler.BasePath, request.Path)); err != nil {
		log.WithField("path", request.Path).WithError(err).Error("DeleteFile failed")
		return err
	}

	return nil
}

func (handler *FileboxMessageHandler) WriteFile(request protocol.WriteFileRequest) (*protocol.WriteFileResponse, error) {
	file, ok := handler.fileHandles.Load(request.FileHandle)
	if !ok {
		log.WithField("fh", request.FileHandle).Error("Invalid file handle in WriteFile request")
		return nil, os.ErrInvalid
	}

	log.WithFields(log.Fields{
		"fh":     request.FileHandle,
		"offset": request.Offset,
		"size":   len(request.Data),
	}).Tracef("Writing file %s", file.(*os.File).Name())

	bytesWritten, err := file.(*os.File).WriteAt(request.Data, request.Offset)
	if err != nil && err != io.EOF {
		log.WithFields(log.Fields{
			"fh":     request.FileHandle,
			"offset": request.Offset,
			"size":   len(request.Data),
		}).WithError(err).Error("file.WriteAt failed")
		return nil, err
	}

	return &protocol.WriteFileResponse{
		BytesWritten: bytesWritten,
	}, nil
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
