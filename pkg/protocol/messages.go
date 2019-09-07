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

// Read Directory
type ReadDirectoryRequestMessage struct{ Path string }
type ReadDirectoryResponseMessage struct{ Files []FileInfo }

// Get File Attributes
type GetFileAttributesRequestMessage struct{ Path string }
type GetFileAttributesResponseMessage struct{ FileInfo FileInfo }

func Init() {
	gob.Register(ReadDirectoryRequestMessage{})
	gob.Register(ReadDirectoryResponseMessage{})
	gob.Register(GetFileAttributesRequestMessage{})
	gob.Register(GetFileAttributesResponseMessage{})
}
