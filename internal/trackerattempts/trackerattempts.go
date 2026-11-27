// Package trackerattempts is the instrumented mutation interface for tracker
// writes made by a dispatched host. The shark CLI appends one entry per
// invocation to the run's ledger before it parses or validates anything, so an
// attempt that is rejected before any mutation is still on record. The parent
// reads the ledger after the dispatch and binds it to the review run.
//
// Trust model: the ledger is evidence about an honest host. It records every
// invocation of the shark CLI that inherited the run's environment, including
// ones rejected before they mutate. A host that bypasses the CLI, clears the
// environment or rewrites the file is outside what it can detect, and an
// invocation's argv names a finding only by the text the host typed.
package trackerattempts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	// EnvLog names the ledger file a dispatched host's shark invocations append to.
	EnvLog = "SHARK_TRACKER_ATTEMPT_LOG"
	// EnvRun names the review run the ledger entries belong to.
	EnvRun = "SHARK_TRACKER_ATTEMPT_RUN"
	// LedgerName is the ledger file name inside the run directory.
	LedgerName = "tracker-attempts.jsonl"
)

// Entry is one shark invocation attempted by the host.
type Entry struct {
	RunID   string   `json:"run_id"`
	Time    string   `json:"time"`
	Command string   `json:"command"`
	Argv    []string `json:"argv"`
	// Write is true for every invocation not known to be read-only.
	Write bool `json:"write"`
}

// Append records an attempt with a single O_APPEND write so concurrent shark
// invocations never interleave a line.
func Append(path, runID, command string, argv []string, write bool) error {
	line, err := json.Marshal(Entry{RunID: runID, Time: time.Now().UTC().Format(time.RFC3339Nano), Command: command, Argv: argv, Write: write})
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

// Ledger is a decoded, digest-bound ledger.
type Ledger struct {
	Entries []Entry
	SHA256  string
}

// Parse decodes a ledger and rejects any entry that is not bound to runID.
func Parse(data []byte, runID string) (Ledger, error) {
	sum := sha256.Sum256(data)
	ledger := Ledger{SHA256: hex.EncodeToString(sum[:])}
	for i, line := range bytes.Split(data, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			return Ledger{}, fmt.Errorf("tracker attempt ledger line %d is invalid: %w", i+1, err)
		}
		if entry.RunID != runID {
			return Ledger{}, fmt.Errorf("tracker attempt ledger line %d belongs to run %q, not %q", i+1, entry.RunID, runID)
		}
		ledger.Entries = append(ledger.Entries, entry)
	}
	return ledger, nil
}

// WriteAttempts counts write attempts that concern findingID. An attempt that
// names another finding in allIDs belongs to that finding. One that names no
// finding is charged to findingID unless claimedElsewhere reports that another
// finding of the run is expected to write to the tracker.
func (l Ledger) WriteAttempts(findingID string, allIDs []string, claimedElsewhere bool) int {
	count := 0
	for _, entry := range l.Entries {
		if !entry.Write {
			continue
		}
		text := strings.Join(entry.Argv, " ")
		switch {
		case mentions(text, findingID):
			count++
		case mentionsAny(text, allIDs):
		case !claimedElsewhere:
			count++
		}
	}
	return count
}

func mentionsAny(text string, ids []string) bool {
	for _, id := range ids {
		if id != "" && mentions(text, id) {
			return true
		}
	}
	return false
}

// mentions reports whether text names id as a whole token, so F-1 does not
// match F-10.
func mentions(text, id string) bool {
	if id == "" {
		return false
	}
	for from := 0; ; {
		at := strings.Index(text[from:], id)
		if at < 0 {
			return false
		}
		start, end := from+at, from+at+len(id)
		if (start == 0 || !idRune(text[start-1])) && (end == len(text) || !idRune(text[end])) {
			return true
		}
		from = start + 1
	}
}

func idRune(b byte) bool {
	return b == '-' || b == '_' || (b >= '0' && b <= '9') || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}
