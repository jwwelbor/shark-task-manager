package config

import "time"

// MaintainerConfig holds the configuration for the maintainer authorization gate.
// It is stored as the "maintainer" object in .sharkconfig.json.
// An absent or nil MaintainerConfig is equivalent to "no password configured,"
// which causes Authorize to fail with an *UnauthorizedError directing the user
// to run `shark admin maintainer set-password`.
//
// Spec reference: spec.md §2.3, REQ-F-007.
type MaintainerConfig struct {
	// PasswordHash is the SHA-256 hex digest of the maintainer password.
	// An empty string means no password has been configured.
	PasswordHash string `json:"password_hash,omitempty"`

	// CacheWindowSeconds is the duration in seconds for which a successful
	// authorization is cached (sudo-style). A zero or negative value uses
	// the default of 60 seconds.
	CacheWindowSeconds int `json:"cache_window_seconds,omitempty"`
}

// CacheWindow returns the configured cache window as a time.Duration.
// When the receiver is nil, or when CacheWindowSeconds is 0 or negative,
// it returns the default of 60 seconds.
//
// This is the source of the default window value for FileGate construction.
func (m *MaintainerConfig) CacheWindow() time.Duration {
	if m == nil || m.CacheWindowSeconds <= 0 {
		return 60 * time.Second
	}
	return time.Duration(m.CacheWindowSeconds) * time.Second
}
