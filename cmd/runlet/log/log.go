package log

import (
	"io"
	"os"
)

type Middleware func(p []byte) (int, error)

func (mw Middleware) Write(p []byte) (int, error) {
	return mw(p)
}

func (mw Middleware) Chain(wr io.Writer) Middleware {
	return func(p []byte) (int, error) {
		var buf []byte
		copy(p, buf)

		_, err := mw.Write(buf)
		if err != nil {
			return -1, err
		}

		return wr.Write(p)
	}
}

// File is a log backend backed by a file on the host.
type File struct {
	f *os.File
}

// NewFile returns a new file task log at the given path.
func NewFile(path string) (*File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	return &File{f: f}, nil
}
