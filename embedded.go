// Package sharktaskmanager provides embedded assets for the Shark Task Manager.
// This file exists at the project root so that go:embed can access shark-templates/
// which lives alongside go.mod. Internal packages import this to avoid duplicating
// the template files.
package sharktaskmanager

import "embed"

// EmbeddedSharkTemplates contains the entire shark-templates/ directory tree,
// including underscore-prefixed partials (e.g., _read_section.tmpl).
// The "all:" prefix ensures files starting with _ or . are included.
//
//go:embed all:shark-templates
var EmbeddedSharkTemplates embed.FS
