package domain

import "bufio"

type Stream struct {
	Data chan StreamData
}

type StreamData struct {
	ID             string
	BufferedReader *bufio.Reader
}
