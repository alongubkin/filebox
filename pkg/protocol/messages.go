package protocol

import (
	"encoding/gob"
	"os"
	"time"
)

type Message struct {
	MessageID  uint32
	IsResponse bool
	Data       interface{}
	Success    bool
}

type EmptyResponse struct{}

type FileInfo struct {
	Name    string      // base name of the file
	Size    int64       // length in bytes for regular files; system-dependent for others
	Mode    os.FileMode // file mode bits
	ModTime time.Time   // modification time
	IsDir   bool        // abbreviation for Mode().IsDir()
}

type OpenFileRequest struct {
	Path  string
	Flags int
}

type OpenFileResponse struct {
	FileHandle uint64
}

type ReadFileRequest struct {
	FileHandle uint64
	Offset     int64
	Size       int
}

type ReadFileResponse struct {
	Data      []byte
	BytesRead int
}

type ReadDirectoryRequest struct {
	Path string
}

type ReadDirectoryResponse struct {
	Files []FileInfo
}

type GetFileAttributesRequest struct {
	Path       string
	FileHandle uint64
}

type GetFileAttributesResponse struct {
	FileInfo FileInfo
}

type CloseFileRequest struct {
	FileHandle uint64
}

type CreateDirectoryRequest struct {
	Path string
	Mode uint32
}

type CreateFileRequest struct {
	Path string
}

type RenameRequest struct {
	OldPath string
	NewPath string
}

type DeleteDirectoryRequest struct {
	Path string
}

type TruncateRequest struct {
	Path       string
	FileHandle uint64
	Size       int64
}

type DeleteFileRequest struct {
	Path string
}

type WriteFileRequest struct {
	FileHandle uint64
	Offset     int64
	Data       []byte
}

type WriteFileResponse struct {
	BytesWritten int
}

func Init() {
	gob.Register(EmptyResponse{})
	gob.Register(OpenFileRequest{})
	gob.Register(OpenFileResponse{})
	gob.Register(ReadFileRequest{})
	gob.Register(ReadFileResponse{})
	gob.Register(ReadDirectoryRequest{})
	gob.Register(ReadDirectoryResponse{})
	gob.Register(GetFileAttributesRequest{})
	gob.Register(GetFileAttributesResponse{})
	gob.Register(CloseFileRequest{})
	gob.Register(CreateDirectoryRequest{})
	gob.Register(CreateFileRequest{})
	gob.Register(RenameRequest{})
	gob.Register(DeleteDirectoryRequest{})
	gob.Register(TruncateRequest{})
	gob.Register(DeleteFileRequest{})
	gob.Register(WriteFileRequest{})
	gob.Register(WriteFileResponse{})
}
