// Package testcapture provides safe process-stream capture primitives for
// tests that need to read captured output after the command finishes.
package testcapture

import (
	"bytes"
	"io"
	"os"
	"sync"
)

// NewDrainingPipe returns a reader for captured output and a writer that can
// be assigned to os.Stdout or os.Stderr. It starts draining the pipe before
// returning, so the producer cannot block when it writes more than the
// operating system's pipe buffer. Close the writer before reading the result.
//
// This is for post-run output assertions. Tests that must observe output while
// the producer is still running should use an explicit streaming reader.
func NewDrainingPipe() (io.ReadCloser, *os.File, error) {
	pipeReader, writer, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}

	var captured bytes.Buffer
	copyDone := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(&captured, pipeReader)
		closeErr := pipeReader.Close()
		if copyErr == nil {
			copyErr = closeErr
		}
		copyDone <- copyErr
	}()

	return &drainedReader{
		buffer: &captured,
		done:   copyDone,
	}, writer, nil
}

type drainedReader struct {
	buffer *bytes.Buffer
	done   <-chan error

	once      sync.Once
	reader    *bytes.Reader
	copyError error
}

func (r *drainedReader) waitForDrain() {
	r.once.Do(func() {
		r.copyError = <-r.done
		r.reader = bytes.NewReader(r.buffer.Bytes())
	})
}

func (r *drainedReader) Read(p []byte) (int, error) {
	r.waitForDrain()
	if r.copyError != nil {
		return 0, r.copyError
	}
	return r.reader.Read(p)
}

func (r *drainedReader) Close() error {
	r.waitForDrain()
	return r.copyError
}
