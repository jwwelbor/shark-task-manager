package commands

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Mock for MaintainerBootstrapService — function-field pattern per
// .claude/rules/services/testing.md.
// ---------------------------------------------------------------------------

// mockMaintainerBootstrapServiceImpl is a test double for maintainerBootstrapServiceIface.
type mockMaintainerBootstrapServiceImpl struct {
	SetPasswordFunc func(ctx context.Context, plaintextPassword string) error
}

func (m *mockMaintainerBootstrapServiceImpl) SetPassword(ctx context.Context, plaintextPassword string) error {
	if m.SetPasswordFunc != nil {
		return m.SetPasswordFunc(ctx, plaintextPassword)
	}
	return fmt.Errorf("SetPassword not implemented in mock")
}

// sha256hexForTest computes the SHA-256 hex digest of a string.
// Used to verify the hash doesn't appear in output (AC-12).
func sha256hexForTest(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

// buildSetPasswordCmdWithMock creates a fresh cobra command tree with an injected
// bootstrap service for testing. It wires the real runAdminMaintainerSetPassword
// logic but substitutes the service.
func buildSetPasswordCmdWithMock(svc maintainerBootstrapServiceIface) *cobra.Command {
	root := &cobra.Command{Use: "shark"}
	admin := &cobra.Command{Use: "admin"}
	maint := &cobra.Command{Use: "maintainer"}
	root.AddCommand(admin)
	admin.AddCommand(maint)

	setPasswordCmd := newAdminMaintainerSetPasswordCmd(svc)
	maint.AddCommand(setPasswordCmd)

	return root
}

// ---------------------------------------------------------------------------
// AC-11 (CLI half): set-password passes the correct password to the service
// ---------------------------------------------------------------------------

func TestAdminMaintainerSetPassword_WritesHashPreservesOtherFields(t *testing.T) {
	password := "hunter2"
	var capturedPassword string
	called := false

	mock := &mockMaintainerBootstrapServiceImpl{
		SetPasswordFunc: func(ctx context.Context, pwd string) error {
			capturedPassword = pwd
			called = true
			return nil
		},
	}

	root := buildSetPasswordCmdWithMock(mock)
	root.SetArgs([]string{"admin", "maintainer", "set-password", "--password", password})

	err := root.Execute()
	if err != nil {
		t.Fatalf("Execute() returned unexpected error: %v", err)
	}

	if !called {
		t.Fatal("expected SetPassword to be called on the mock service, but it was not")
	}

	if capturedPassword != password {
		t.Errorf("SetPassword called with password %q, want %q", capturedPassword, password)
	}
}

// ---------------------------------------------------------------------------
// AC-12: set-password prints no password or hash on stdout/stderr
// ---------------------------------------------------------------------------

func TestAdminMaintainerSetPassword_NoSecretInOutput(t *testing.T) {
	password := "hunter2"
	expectedHash := sha256hexForTest(password)

	mock := &mockMaintainerBootstrapServiceImpl{
		SetPasswordFunc: func(ctx context.Context, pwd string) error {
			return nil
		},
	}

	// Use cobra's SetOut/SetErr to capture output without touching OS streams.
	var outBuf, errBuf bytes.Buffer
	root := buildSetPasswordCmdWithMock(mock)
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs([]string{"admin", "maintainer", "set-password", "--password", password})

	_ = root.Execute()

	allOutput := outBuf.String() + errBuf.String()

	if strings.Contains(allOutput, password) {
		t.Errorf("plaintext password %q found in output: %s", password, allOutput)
	}

	if strings.Contains(allOutput, expectedHash) {
		t.Errorf("password hash %q found in output: %s", expectedHash, allOutput)
	}
}

// ---------------------------------------------------------------------------
// AC-12: Error path — service error does not leak the password
// ---------------------------------------------------------------------------

func TestAdminMaintainerSetPassword_ServiceError_DoesNotLeakPassword(t *testing.T) {
	password := "secret-password-xyz"
	serviceErr := errors.New("config write failed")

	mock := &mockMaintainerBootstrapServiceImpl{
		SetPasswordFunc: func(ctx context.Context, pwd string) error {
			return serviceErr
		},
	}

	var outBuf, errBuf bytes.Buffer
	root := buildSetPasswordCmdWithMock(mock)
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs([]string{"admin", "maintainer", "set-password", "--password", password})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error from service failure, got nil")
	}

	allOutput := outBuf.String() + errBuf.String()
	if strings.Contains(allOutput, password) {
		t.Errorf("plaintext password %q found in error output: %s", password, allOutput)
	}
}

// ---------------------------------------------------------------------------
// AC-T1: --password-stdin flag reads password from stdin
// ---------------------------------------------------------------------------

func TestAdminMaintainerSetPassword_StdinFlag(t *testing.T) {
	password := "from-stdin"
	var capturedPassword string

	mock := &mockMaintainerBootstrapServiceImpl{
		SetPasswordFunc: func(ctx context.Context, pwd string) error {
			capturedPassword = pwd
			return nil
		},
	}

	// Provide stdin input via a pipe
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	fmt.Fprintln(w, password)
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	root := buildSetPasswordCmdWithMock(mock)
	root.SetArgs([]string{"admin", "maintainer", "set-password", "--password-stdin"})

	execErr := root.Execute()
	if execErr != nil {
		t.Fatalf("Execute() returned unexpected error: %v", execErr)
	}

	if capturedPassword != password {
		t.Errorf("password from stdin = %q, want %q", capturedPassword, password)
	}
}
