package pathutil

import (
	"path/filepath"
	"testing"
)

func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)        // unix: os.UserHomeDir reads $HOME
	t.Setenv("USERPROFILE", home) // windows parity

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"tilde-slash expands", "~/bundle/content", filepath.Join(home, "bundle", "content")},
		{"bare tilde not expanded", "~bundle", "~bundle"},
		{"relative unchanged", "shark-data", "shark-data"},
		{"absolute unchanged", "/etc/shark", "/etc/shark"},
		{"empty unchanged", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExpandHome(tt.in); got != tt.want {
				t.Errorf("ExpandHome(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
