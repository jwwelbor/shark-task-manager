package gatepersist

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/models"
)

// fakeWorld is an in-memory stand-in for every injected dependency
// (NoteWriter, NoteReader, HistoryReader, Transitioner, LeaseReleaser). It
// lets tests simulate a crash at any point in the persistence sequence: a
// "fail after commit" injection still durably records the write (as a real
// crash would, after the target store's own commit) but returns an error to
// Persist (as if the process died before recording that fact in
// operation-state.json) — the exact crash window REQ-F-003 exists for.
type fakeWorld struct {
	mu sync.Mutex

	notes  []*models.EntityNote
	nextID int64

	// history is keyed by "entityType|entityKey".
	history map[string][]*models.EntityHistory
	// statuses is keyed by "entityType|entityKey".
	statuses map[string]string

	releaseCalls []releaseCall

	// failNoteContent, keyed by "noteType|content", injects a failure after
	// the note is durably appended but before Persist observes success.
	failNoteContent map[string]bool
	// failTransitionTo, keyed by "entityType|entityKey|targetStatus",
	// injects a failure after the status is durably updated.
	failTransitionTo map[string]bool

	transitionCalls []transitionCall
}

type releaseCall struct {
	entityType, entityKey, sessionID, outcome string
	force                                     bool
}

type transitionCall struct {
	entityType, entityKey, targetStatus, reason, agent string
	guard                                              TransitionGuard
}

func newFakeWorld() *fakeWorld {
	return &fakeWorld{
		history:          make(map[string][]*models.EntityHistory),
		statuses:         make(map[string]string),
		failNoteContent:  make(map[string]bool),
		failTransitionTo: make(map[string]bool),
	}
}

func historyKey(entityType models.EntityType, entityKey string) string {
	return string(entityType) + "|" + entityKey
}

func (w *fakeWorld) setStatus(entityType models.EntityType, entityKey, status string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.statuses[historyKey(entityType, entityKey)] = status
}

// AddNoteWithMetadata implements NoteWriter.
func (w *fakeWorld) AddNoteWithMetadata(_ context.Context, entityType models.EntityType, entityKey, noteType, content, createdBy, metadata string) (*models.EntityNote, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.nextID++
	meta := metadata
	note := &models.EntityNote{
		ID:         w.nextID,
		EntityType: entityType,
		Content:    content,
		NoteType:   models.NoteType(noteType),
		CreatedBy:  &createdBy,
		Metadata:   &meta,
	}
	note.EntityID = entityIDFor(entityKey)
	w.notes = append(w.notes, note)

	if w.failNoteContent[noteType+"|"+content] {
		delete(w.failNoteContent, noteType+"|"+content) // fails exactly once, like a real one-time crash
		return nil, fmt.Errorf("fakeWorld: injected failure writing %s note", noteType)
	}
	return note, nil
}

// ListNotes implements NoteReader. entityKey is resolved back from the
// synthetic entity ID this fake assigns, since EntityNote only stores
// EntityID (matching the real repository's shape).
func (w *fakeWorld) ListNotes(_ context.Context, entityType models.EntityType, entityKey string, _ []string) ([]*models.EntityNote, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	id := entityIDFor(entityKey)
	var out []*models.EntityNote
	for _, n := range w.notes {
		if n.EntityType == entityType && n.EntityID == id {
			out = append(out, n)
		}
	}
	return out, nil
}

// GetHistory implements HistoryReader.
func (w *fakeWorld) GetHistory(_ context.Context, entityType models.EntityType, entityKey string) ([]*models.EntityHistory, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]*models.EntityHistory(nil), w.history[historyKey(entityType, entityKey)]...), nil
}

// Transition implements Transitioner, mirroring EntityService.
// TransitionStatus's own idempotency guarantee: calling it again when
// already at targetStatus is a no-op success (transitioned=false).
func (w *fakeWorld) Transition(_ context.Context, entityType models.EntityType, entityKey, targetStatus, reason, agent string, guard TransitionGuard) (string, bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.transitionCalls = append(w.transitionCalls, transitionCall{
		entityType: string(entityType), entityKey: entityKey, targetStatus: targetStatus, reason: reason, agent: agent, guard: guard,
	})

	key := historyKey(entityType, entityKey)
	from := w.statuses[key]
	if from == "" {
		from = "todo"
	}
	if from == targetStatus {
		return from, false, nil
	}

	w.statuses[key] = targetStatus
	notes := reason
	w.history[key] = append(w.history[key], &models.EntityHistory{
		EntityType: entityType,
		FromStatus: &from,
		ToStatus:   targetStatus,
		Notes:      &notes,
	})

	failKey := string(entityType) + "|" + entityKey + "|" + targetStatus
	if w.failTransitionTo[failKey] {
		delete(w.failTransitionTo, failKey)
		return "", false, fmt.Errorf("fakeWorld: injected failure transitioning %s %s to %s", entityType, entityKey, targetStatus)
	}
	return from, true, nil
}

// CurrentStatus implements StatusReader.
func (w *fakeWorld) CurrentStatus(_ context.Context, entityType models.EntityType, entityKey string) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if s, ok := w.statuses[historyKey(entityType, entityKey)]; ok {
		return s, nil
	}
	return "todo", nil
}

// Release implements LeaseReleaser.
func (w *fakeWorld) Release(_ context.Context, entityType, entityKey, sessionID, outcome string, force bool) (bool, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.releaseCalls = append(w.releaseCalls, releaseCall{entityType: entityType, entityKey: entityKey, sessionID: sessionID, outcome: outcome, force: force})
	return true, nil
}

// entityIDFor derives a stable synthetic numeric ID from an entity key, so
// EntityNote.EntityID round-trips consistently without a real registry.
func entityIDFor(entityKey string) int64 {
	var sum int64
	for _, r := range entityKey {
		sum = sum*31 + int64(r)
	}
	if sum < 0 {
		sum = -sum
	}
	return sum + 1
}

// fakeStatusValidator implements StatusValidator with a fixed allowlist per
// entity type.
type fakeStatusValidator struct {
	valid map[models.EntityType]map[string]bool
}

func newFakeStatusValidator() *fakeStatusValidator {
	return &fakeStatusValidator{valid: make(map[models.EntityType]map[string]bool)}
}

func (v *fakeStatusValidator) allow(entityType models.EntityType, statuses ...string) *fakeStatusValidator {
	if v.valid[entityType] == nil {
		v.valid[entityType] = make(map[string]bool)
	}
	for _, s := range statuses {
		v.valid[entityType][s] = true
	}
	return v
}

func (v *fakeStatusValidator) IsValidStatus(entityType models.EntityType, status string) bool {
	return v.valid[entityType][status]
}

// fakeClaimVerifier implements gatepersist.ClaimVerifier for the UAT
// round-2 Finding 1 (TOCTOU) tests: getCalls records how many times Get was
// invoked, and each entry in claims is returned in order, one per call
// (with the last entry reused for any call beyond len(claims)) -- this is
// what lets a test simulate "the claim was still valid when run.go's
// verifyClaimSession checked it, but expired/was reclaimed by the time
// Persist's own re-check ran": the first Get() (the CLI-level check) and
// the second Get() (Persist's re-check) return different claim states.
type fakeClaimVerifier struct {
	mu       sync.Mutex
	ttl      time.Duration
	claims   []*models.EntityClaim
	getCalls int
}

func (v *fakeClaimVerifier) Get(_ context.Context, _, _ string) (*models.EntityClaim, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	idx := v.getCalls
	if idx >= len(v.claims) {
		idx = len(v.claims) - 1
	}
	v.getCalls++
	if idx < 0 {
		return nil, nil
	}
	return v.claims[idx], nil
}

func (v *fakeClaimVerifier) TTL() time.Duration {
	if v.ttl > 0 {
		return v.ttl
	}
	return time.Hour
}
