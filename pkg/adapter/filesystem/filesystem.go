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
	Open(path string) (ClosableRWFile, error)
	OpenFile(path string, flag int, perm os.FileMode) (ClosableRWFile, error)
	ReadAll(r io.Reader) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	Remove(name string) error
	RemoveAll(path string) error
	ReadDir(name string) ([]os.DirEntry, error)
	Copy(dst io.Writer, src io.Reader) (written int64, err error)
	WalkDir(root string, fn fs.WalkDirFunc) error
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

func (f FileSystem) Open(path string) (ClosableRWFile, error) {
	return os.Open(path)
}

func (f FileSystem) Copy(dst io.Writer, src io.Reader) (written int64, err error) {
	return io.Copy(dst, src)
}

func (f FileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	return f.WalkDir(root, fn)
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

func (f FileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (f FileSystem) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}
