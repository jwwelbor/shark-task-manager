// Package viewer provides the embedded single-file SPA for the shark status
// viewer dashboard. The HTML file is embedded at build time via go:embed so
// that the binary ships with no external file dependencies.
package viewer

import _ "embed"

//go:embed assets/viewer.html
var ViewerHTML []byte
