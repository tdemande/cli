package net

import (
	"fmt"
	"io"
	"os"

	"github.com/cloudfoundry/cli/cf/formatters"
)

type ProgressReader struct {
	ioReadSeeker io.ReadSeeker
	red          int64
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

	fmt.Println("read", n, " bytes")
	if err != nil {
		fmt.Println(err)
	}

	if progressReader.total > int64(0) {
		progressReader.red += int64(n)

		if progressReader.total == progressReader.red {
			fmt.Println(" " + "Upload complete.")
			return n, err
		}

		if err == nil {
			fmt.Printf("\r%s uploaded.", formatters.ByteSize(progressReader.red))
		}
	}

	return n, err
}

func (progressReader *ProgressReader) Seek(offset int64, whence int) (int64, error) {
	return progressReader.ioReadSeeker.Seek(offset, whence)
}
