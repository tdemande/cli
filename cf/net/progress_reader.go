package net

import (
	"fmt"
	"io"
	"os"
)

type ProgressReader struct {
	ioReadSeeker io.ReadSeeker
	read         int64
	total        int64
}

func NewProgressReader(readSeeker io.ReadSeeker) *ProgressReader {
	return &ProgressReader{
		ioReadSeeker: readSeeker,
	}
}

func (progressReader *ProgressReader) Read(p []byte) (int, error) {
	if progressReader.ioReadSeeker == nil {
		return 0, os.ErrInvalid
	}

	n, err := progressReader.ioReadSeeker.Read(p)

	if progressReader.total > int64(0) {
		progressReader.read += int64(n)

		if err == nil {
			fmt.Printfi("\x0c%dK uploaded", progressReader.read/int64(1024))
		}
	}

	return n, err
}

func (progressReader *ProgressReader) Seek(offset int64, whence int) (int64, error) {
	return progressReader.ioReadSeeker.Seek(offset, whence)
}
