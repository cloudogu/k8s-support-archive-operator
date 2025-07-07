package filesystem

import (
	"io"
	"io/fs"
	"os"
)

type ClosableRWFile interface {
	Close() error
	Write(p []byte) (n int, err error)
	Read(p []byte) (n int, err error)
}

type Filesystem interface {
	Stat(name string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
	Create(name string) (ClosableRWFile, error)
	OpenFile(path string, flag int, perm os.FileMode) (ClosableRWFile, error)
	ReadAll(r io.Reader) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	Remove(name string) error
	ReadDir(name string) ([]os.DirEntry, error)
}

type FileSystem struct{}

func (f FileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (f FileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (f FileSystem) Create(name string) (ClosableRWFile, error) {
	return os.Create(name)
}

func (f FileSystem) OpenFile(path string, flag int, perm os.FileMode) (ClosableRWFile, error) {
	return os.OpenFile(path, flag, perm)
}

func (f FileSystem) ReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

func (f FileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (f FileSystem) Remove(path string) error {
	return os.Remove(path)
}

func (f FileSystem) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}
