package main

import (
	"io"
	"os"
)

type stdrwc struct{}

// NewStdIOReadWriteCloser - For reading on stdin and writing on stdout.
func NewStdIOReadWriteCloser() io.ReadWriteCloser {
	return stdrwc{}
}

func (stdrwc) Read(p []byte) (int, error) {
	return os.Stdin.Read(p)
}

func (stdrwc) Write(p []byte) (int, error) {
	return os.Stdout.Write(p)
}

func (stdrwc) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
}
