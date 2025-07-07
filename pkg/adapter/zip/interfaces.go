package zip

import "io"

type Zipper interface {
	Close() error
	Create(name string) (io.Writer, error)
}

type ZipCreator interface {
	NewWriter(w io.Writer) Zipper
}
