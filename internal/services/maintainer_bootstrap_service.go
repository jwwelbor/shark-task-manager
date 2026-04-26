package services

import (
	"context"
	"crypto/sha256"
	"fmt"
)

// ConfigReader is the interface used by MaintainerBootstrapService to read the
// current configuration. An implementation may read from .sharkconfig.json or
// return nil/empty to signal "no config yet."
//
// Returning (nil, nil) is a valid response that means "config absent; start fresh."
// Returning (nil, err) means a hard read failure that should abort the operation.
//
// Spec reference: spec.md §2.5 F02-D4, AC-T1.
type ConfigReader interface {
	// Read loads the raw config map from storage.
	// Returns (nil, nil) when the config file does not exist.
	// Returns (nil, err) on I/O or parse errors.
	Read() (map[string]interface{}, error)
}

// ConfigWriter is the interface used by MaintainerBootstrapService to persist
// the updated configuration.
//
// Spec reference: spec.md §2.5 F02-D4, AC-T1.
type ConfigWriter interface {
	// Write persists the given raw config map to storage.
	Write(data map[string]interface{}) error
}

// MaintainerBootstrapService orchestrates the "read config → compute SHA-256
// hash → write password_hash back to config" flow for the bootstrap command
// `shark admin maintainer set-password`.
//
// It is intentionally free of filesystem concerns: config I/O is delegated to
// the injected ConfigReader and ConfigWriter interfaces, so the service can be
// tested with pure in-memory mocks (no disk access required).
//
// Spec reference: spec.md §2.1, §2.5 F02-D4, REQ-F-008, AC-T1..AC-T5.
type MaintainerBootstrapService struct {
	reader ConfigReader
	writer ConfigWriter
}

// NewMaintainerBootstrapService constructs a MaintainerBootstrapService with
// the given reader and writer. Both are required.
func NewMaintainerBootstrapService(reader ConfigReader, writer ConfigWriter) *MaintainerBootstrapService {
	return &MaintainerBootstrapService{
		reader: reader,
		writer: writer,
	}
}

// SetPassword computes the SHA-256 hex digest of plaintextPassword and writes
// it into the config as maintainer.password_hash, preserving all other
// existing config keys.
//
// Behaviour:
//   - Reads the current config from the injected ConfigReader.
//   - If the reader returns (nil, nil), an empty config map is used.
//   - Computes sha256hex(plaintextPassword).
//   - Updates (or creates) the "maintainer" object with the new password_hash,
//     preserving any existing maintainer keys (e.g. cache_window_seconds).
//   - Writes the updated map via the injected ConfigWriter.
//   - The plaintext password is NEVER stored in any field.
//
// Returns a non-nil error (wrapping the underlying cause) if the read or write
// operation fails. Spec reference: REQ-F-008, AC-T2, AC-T3, AC-T4, AC-T5.
func (s *MaintainerBootstrapService) SetPassword(ctx context.Context, plaintextPassword string) error {
	// Step 1: Read existing config (or start fresh on nil/absent config).
	raw, err := s.reader.Read()
	if err != nil {
		return fmt.Errorf("maintainer bootstrap: failed to read config: %w", err)
	}
	if raw == nil {
		// No config exists yet; start with an empty map.
		raw = make(map[string]interface{})
	}

	// Step 2: Compute SHA-256 hex digest.
	// The plaintext password is used only here and is not stored.
	digest := computeSHA256Hex(plaintextPassword)

	// Step 3: Merge into the "maintainer" sub-object, preserving existing keys.
	maintainer := extractMaintainerMap(raw)
	maintainer["password_hash"] = digest

	raw["maintainer"] = maintainer

	// Step 4: Write updated config.
	if err := s.writer.Write(raw); err != nil {
		return fmt.Errorf("maintainer bootstrap: failed to write config: %w", err)
	}

	return nil
}

// computeSHA256Hex returns the lowercase hex-encoded SHA-256 digest of s.
// This is the only place where the plaintext password is processed; it must
// never be stored in any field.
//
// Spec reference: REQ-F-006, REQ-F-008.
func computeSHA256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

// extractMaintainerMap returns the existing "maintainer" map from raw, or an
// empty map if none exists. This preserves fields like cache_window_seconds.
func extractMaintainerMap(raw map[string]interface{}) map[string]interface{} {
	if existing, ok := raw["maintainer"].(map[string]interface{}); ok {
		// Return a shallow copy to avoid mutating the original during step 3.
		out := make(map[string]interface{}, len(existing))
		for k, v := range existing {
			out[k] = v
		}
		return out
	}
	return make(map[string]interface{})
}
