package commands

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

// TestNextCommandBareRequiresEntityKey pins the restored 0e3f0103 contract:
// bare `shark next` (no entity key) must fail before any portfolio,
// planning, or dispatch service is constructed, and the error must point the
// operator at `shark plan`.
func TestNextCommandBareRequiresEntityKey(t *testing.T) {
	originalAdapterFactory := nextNewAdapterCache
	defer func() { nextNewAdapterCache = originalAdapterFactory }()

	adapterCalls := 0
	nextNewAdapterCache = func(context.Context) (*nextAdapterCache, error) {
		adapterCalls++
		t.Fatal("bare next constructed keyed adapters")
		return nil, nil
	}

	cmd := newNextCommand()
	cmd.SilenceUsage = true
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{})
	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("ExecuteContext() error = nil, want a required-entity-key error")
	}
	if !strings.Contains(err.Error(), "shark next requires an entity key") ||
		!strings.Contains(err.Error(), "shark plan") {
		t.Fatalf("error = %v, want it to require an entity key and mention shark plan", err)
	}
	if adapterCalls != 0 {
		t.Fatalf("adapter calls = %d, want 0", adapterCalls)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no JSON output on rejection", stdout.String())
	}
}

func TestNextCommandArgumentValidationRunsBeforeFactories(t *testing.T) {
	originalAdapterFactory := nextNewAdapterCache
	defer func() { nextNewAdapterCache = originalAdapterFactory }()

	adapterCalls := 0
	nextNewAdapterCache = func(context.Context) (*nextAdapterCache, error) {
		adapterCalls++
		t.Fatal("invalid arguments constructed keyed adapters")
		return nil, nil
	}

	for _, args := range [][]string{{"E36", "extra"}, {"E36", "extra", "third"}} {
		cmd := newNextCommand()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(args)
		err := cmd.ExecuteContext(context.Background())
		if err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s)") {
			t.Errorf("args %v error = %v, want Cobra exact-arguments error", args, err)
		}
	}
	if adapterCalls != 0 {
		t.Fatalf("adapter calls = %d, want 0", adapterCalls)
	}
}

func TestNextCommandPreviewIsUnknownBeforeFactories(t *testing.T) {
	originalAdapterFactory := nextNewAdapterCache
	defer func() { nextNewAdapterCache = originalAdapterFactory }()

	adapterCalls := 0
	nextNewAdapterCache = func(context.Context) (*nextAdapterCache, error) {
		adapterCalls++
		t.Fatal("unknown flag constructed keyed adapters")
		return nil, nil
	}

	for _, args := range [][]string{{"--preview"}, {"E36", "--preview"}} {
		cmd := newNextCommand()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(args)
		err := cmd.ExecuteContext(context.Background())
		if err == nil || !strings.Contains(err.Error(), "unknown flag: --preview") {
			t.Errorf("args %v error = %v, want Cobra unknown-flag error", args, err)
		}
	}
	if adapterCalls != 0 {
		t.Fatalf("adapter calls = %d, want 0", adapterCalls)
	}
	if nextCmd.Flags().Lookup("preview") != nil {
		t.Error("next command unexpectedly exposes preview")
	}
	if taskNextStatusCmd.Flags().Lookup("preview") == nil {
		t.Error("task lifecycle preview flag was removed")
	}
}

func TestNextCommandKeyedDoesNotConstructPortfolioServices(t *testing.T) {
	originalAdapterFactory := nextNewAdapterCache
	defer func() { nextNewAdapterCache = originalAdapterFactory }()

	sentinel := errors.New("keyed adapter sentinel")
	adapterCalls := 0
	nextNewAdapterCache = func(context.Context) (*nextAdapterCache, error) {
		adapterCalls++
		return nil, sentinel
	}

	cmd := newNextCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"E36"})
	err := cmd.ExecuteContext(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("ExecuteContext() error = %v, want keyed adapter sentinel", err)
	}
	if adapterCalls != 1 {
		t.Fatalf("adapter calls = %d, want 1", adapterCalls)
	}
}
