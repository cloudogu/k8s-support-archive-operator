package domain

import (
	"io"
)

type Stream struct {
	Data chan StreamData
}

type StreamData struct {
	ID     string
	Reader io.Reader
}
