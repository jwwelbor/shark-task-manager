// Package commands contains Shark CLI command implementations.
package commands

// maxCascadeDepth bounds how deep cascade recursion can go. The hierarchy
// is Epic → Feature → Task (depth 2), so a small bound is sufficient.
// Anything deeper indicates a misconfigured workflow YAML and should fail
// loudly rather than spinning.
const maxCascadeDepth = 4
