package domain

import (
	"io"
)

type Stream struct {
	Data chan StreamData
}

type StreamData struct {
	ID                string
	StreamConstructor StreamConstructor
}

type StreamConstructor func() (io.Reader, CloseStreamFunc, error)

type CloseStreamFunc func() error
