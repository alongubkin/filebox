package server

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/alongubkin/filebox/pkg/protocol"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type FileboxMessageHandler struct {
	BasePath string

	// FUTURE: Automatically close handles if their client is disconnected.
	fileHandles    sync.Map
	nextFileHandle uint64
}

func (handler *FileboxMessageHandler) OpenFile(request protocol.OpenFileRequest) (*protocol.OpenFileResponse, error) {
	file, err := openFile(path.Join(handler.BasePath, request.Path),
		request.Flags & ^os.O_EXCL, 00777)
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

// ==========================================================
// All the code below is copied from github.com/moby/moby (file_windows.go), which is in turn
// copied from golang itself. The purpose is to support FILE_SHARE_DELETE on windows.
// ==========================================================
func openFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	if name == "" {
		return nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
	}
	h, err := syscallOpen(fixLongPath(name), flag|syscall.O_CLOEXEC, syscallMode(perm))
	if err != nil {
		return nil, errors.Wrap(err, "error opening file")
	}
	return os.NewFile(uintptr(h), name), nil
}

// fixLongPath returns the extended-length (\\?\-prefixed) form of
// path when needed, in order to avoid the default 260 character file
// path limit imposed by Windows. If path is not easily converted to
// the extended-length form (for example, if path is a relative path
// or contains .. elements), or is short enough, fixLongPath returns
// path unmodified.
//
// See https://msdn.microsoft.com/en-us/library/windows/desktop/aa365247(v=vs.85).aspx#maxpath
//
// Copied from os.OpenFile
func fixLongPath(path string) string {
	// Do nothing (and don't allocate) if the path is "short".
	// Empirically (at least on the Windows Server 2013 builder),
	// the kernel is arbitrarily okay with < 248 bytes. That
	// matches what the docs above say:
	// "When using an API to create a directory, the specified
	// path cannot be so long that you cannot append an 8.3 file
	// name (that is, the directory name cannot exceed MAX_PATH
	// minus 12)." Since MAX_PATH is 260, 260 - 12 = 248.
	//
	// The MSDN docs appear to say that a normal path that is 248 bytes long
	// will work; empirically the path must be less then 248 bytes long.
	if len(path) < 248 {
		// Don't fix. (This is how Go 1.7 and earlier worked,
		// not automatically generating the \\?\ form)
		return path
	}

	// The extended form begins with \\?\, as in
	// \\?\c:\windows\foo.txt or \\?\UNC\server\share\foo.txt.
	// The extended form disables evaluation of . and .. path
	// elements and disables the interpretation of / as equivalent
	// to \. The conversion here rewrites / to \ and elides
	// . elements as well as trailing or duplicate separators. For
	// simplicity it avoids the conversion entirely for relative
	// paths or paths containing .. elements. For now,
	// \\server\share paths are not converted to
	// \\?\UNC\server\share paths because the rules for doing so
	// are less well-specified.
	if len(path) >= 2 && path[:2] == `\\` {
		// Don't canonicalize UNC paths.
		return path
	}
	if !isAbs(path) {
		// Relative path
		return path
	}

	const prefix = `\\?`

	pathbuf := make([]byte, len(prefix)+len(path)+len(`\`))
	copy(pathbuf, prefix)
	n := len(path)
	r, w := 0, len(prefix)
	for r < n {
		switch {
		case os.IsPathSeparator(path[r]):
			// empty block
			r++
		case path[r] == '.' && (r+1 == n || os.IsPathSeparator(path[r+1])):
			// /./
			r++
		case r+1 < n && path[r] == '.' && path[r+1] == '.' && (r+2 == n || os.IsPathSeparator(path[r+2])):
			// /../ is currently unhandled
			return path
		default:
			pathbuf[w] = '\\'
			w++
			for ; r < n && !os.IsPathSeparator(path[r]); r++ {
				pathbuf[w] = path[r]
				w++
			}
		}
	}
	// A drive's root directory needs a trailing \
	if w == len(`\\?\c:`) {
		pathbuf[w] = '\\'
		w++
	}
	return string(pathbuf[:w])
}

func isAbs(path string) (b bool) {
	v := volumeName(path)
	if v == "" {
		return false
	}
	path = path[len(v):]
	if path == "" {
		return false
	}
	return os.IsPathSeparator(path[0])
}

func volumeName(path string) (v string) {
	if len(path) < 2 {
		return ""
	}
	// with drive letter
	c := path[0]
	if path[1] == ':' &&
		('0' <= c && c <= '9' || 'a' <= c && c <= 'z' ||
			'A' <= c && c <= 'Z') {
		return path[:2]
	}
	// is it UNC
	if l := len(path); l >= 5 && os.IsPathSeparator(path[0]) && os.IsPathSeparator(path[1]) &&
		!os.IsPathSeparator(path[2]) && path[2] != '.' {
		// first, leading `\\` and next shouldn't be `\`. its server name.
		for n := 3; n < l-1; n++ {
			// second, next '\' shouldn't be repeated.
			if os.IsPathSeparator(path[n]) {
				n++
				// third, following something characters. its share name.
				if !os.IsPathSeparator(path[n]) {
					if path[n] == '.' {
						break
					}
					for ; n < l; n++ {
						if os.IsPathSeparator(path[n]) {
							break
						}
					}
					return path[:n]
				}
				break
			}
		}
	}
	return ""
}

// copied from os package for os.OpenFile
func syscallMode(i os.FileMode) (o uint32) {
	o |= uint32(i.Perm())
	if i&os.ModeSetuid != 0 {
		o |= syscall.S_ISUID
	}
	if i&os.ModeSetgid != 0 {
		o |= syscall.S_ISGID
	}
	if i&os.ModeSticky != 0 {
		o |= syscall.S_ISVTX
	}
	// No mapping for Go's ModeTemporary (plan9 only).
	return
}

// syscallOpen is copied from syscall.Open but is modified to
// always open a file with FILE_SHARE_DELETE
func syscallOpen(path string, mode int, perm uint32) (fd syscall.Handle, err error) {
	if len(path) == 0 {
		return syscall.InvalidHandle, syscall.ERROR_FILE_NOT_FOUND
	}

	pathp, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return syscall.InvalidHandle, err
	}
	var access uint32
	switch mode & (syscall.O_RDONLY | syscall.O_WRONLY | syscall.O_RDWR) {
	case syscall.O_RDONLY:
		access = syscall.GENERIC_READ
	case syscall.O_WRONLY:
		access = syscall.GENERIC_WRITE
	case syscall.O_RDWR:
		access = syscall.GENERIC_READ | syscall.GENERIC_WRITE
	}
	if mode&syscall.O_CREAT != 0 {
		access |= syscall.GENERIC_WRITE
	}
	if mode&syscall.O_APPEND != 0 {
		access &^= syscall.GENERIC_WRITE
		access |= syscall.FILE_APPEND_DATA
	}
	sharemode := uint32(syscall.FILE_SHARE_READ | syscall.FILE_SHARE_WRITE | syscall.FILE_SHARE_DELETE)
	var sa *syscall.SecurityAttributes
	if mode&syscall.O_CLOEXEC == 0 {
		sa = makeInheritSa()
	}
	var createmode uint32
	switch {
	case mode&(syscall.O_CREAT|syscall.O_EXCL) == (syscall.O_CREAT | syscall.O_EXCL):
		createmode = syscall.CREATE_NEW
	case mode&(syscall.O_CREAT|syscall.O_TRUNC) == (syscall.O_CREAT | syscall.O_TRUNC):
		createmode = syscall.CREATE_ALWAYS
	case mode&syscall.O_CREAT == syscall.O_CREAT:
		createmode = syscall.OPEN_ALWAYS
	case mode&syscall.O_TRUNC == syscall.O_TRUNC:
		createmode = syscall.TRUNCATE_EXISTING
	default:
		createmode = syscall.OPEN_EXISTING
	}
	h, e := syscall.CreateFile(pathp, access, sharemode, sa, createmode, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	return h, e
}

func makeInheritSa() *syscall.SecurityAttributes {
	var sa syscall.SecurityAttributes
	sa.Length = uint32(unsafe.Sizeof(sa))
	sa.InheritHandle = 1
	return &sa
}
