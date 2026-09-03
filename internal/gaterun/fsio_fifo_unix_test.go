//go:build !windows

package gaterun

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// This file holds the FIFO-target regression tests split out of
// fsio_test.go: syscall.Mkfifo has no Windows implementation, so these tests
// (unlike the rest of fsio_test.go, which runs on every platform) are
// POSIX-only and must live behind their own build constraint rather than
// making the whole shared test file — and the internal/gaterun package it
// compiles with — fail `go vet`/`go build` on GOOS=windows.

func TestReadResult_RejectsFIFOTarget(t *testing.T) {
	dir := newRunDir(t)
	path := filepath.Join(dir, resultFileName)
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, _, err := ReadResult(dir); err == nil {
			t.Error("ReadResult over a FIFO target: want error, got nil")
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("ReadResult over a FIFO target blocked instead of returning an error")
	}
}

func TestCreateResult_RejectsFIFOTarget(t *testing.T) {
	dir := newRunDir(t)
	path := filepath.Join(dir, resultFileName)
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}
	if _, err := CreateResult(dir, []byte(`{"a":1}`)); err == nil {
		t.Fatal("CreateResult over a FIFO target: want error, got nil")
	}
}

func TestWriteOperationState_RejectsFIFOTarget(t *testing.T) {
	dir := newRunDir(t)
	path := filepath.Join(dir, operationStateFileName)
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}
	if err := WriteOperationState(dir, []byte(`{"phase":"x"}`)); err == nil {
		t.Fatal("WriteOperationState over a FIFO target: want error, got nil")
	}
}
