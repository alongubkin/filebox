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
	Error      error
}

type FileInfo struct {
	Name    string      // base name of the file
	Size    int64       // length in bytes for regular files; system-dependent for others
	Mode    os.FileMode // file mode bits
	ModTime time.Time   // modification time
	IsDir   bool        // abbreviation for Mode().IsDir()
}

type OpenFileRequestMessage struct {
	Path  string
	Flags int
}

type OpenFileResponseMessage struct {
	FileHandle uint64
}

type ReadFileRequestMessage struct {
	FileHandle uint64
	Offset     int64
	Size       int
}

type ReadFileResponseMessage struct {
	Data      []byte
	BytesRead int
}

type ReadDirectoryRequestMessage struct {
	Path string
}

type ReadDirectoryResponseMessage struct {
	Files []FileInfo
}

type GetFileAttributesRequestMessage struct {
	Path       string
	FileHandle uint64
}

type GetFileAttributesResponseMessage struct {
	FileInfo FileInfo
}

func Init() {
	gob.Register(OpenFileRequestMessage{})
	gob.Register(OpenFileResponseMessage{})
	gob.Register(ReadFileRequestMessage{})
	gob.Register(ReadFileResponseMessage{})
	gob.Register(ReadDirectoryRequestMessage{})
	gob.Register(ReadDirectoryResponseMessage{})
	gob.Register(GetFileAttributesRequestMessage{})
	gob.Register(GetFileAttributesResponseMessage{})
}
