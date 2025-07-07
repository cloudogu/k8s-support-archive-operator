package zip

import (
	"archive/zip"
	"io"
)

type Creator struct{}

func NewCreator() *Creator {
	return &Creator{}
}

func (c *Creator) NewWriter(w io.Writer) Zipper {
	return zip.NewWriter(w)
}
