package observability

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// FileJSONLExporter is an OTel SpanExporter that appends one JSON line per span
// to <projectRoot>/shark-data/.stats/events.jsonl.
//
// It is fail-soft: write errors are logged at debug level and never returned to
// the caller, so a disk-full or permission error never breaks the CLI.
//
// The zero value is safe to use; ExportSpans and Shutdown are both no-ops when
// the path field is empty (mirrors the nil-check pattern in CommandMetrics).
type FileJSONLExporter struct {
	path string // resolved once at construction time; empty means skip writes
}

// NewFileJSONLExporter constructs a FileJSONLExporter whose output file is
// <projectRoot>/shark-data/.stats/events.jsonl.
//
// When projectRoot is empty the exporter is constructed but all writes are
// silently skipped, satisfying the requirement that
// NewFileJSONLExporter("") never crashes.
func NewFileJSONLExporter(projectRoot string) *FileJSONLExporter {
	if projectRoot == "" {
		return &FileJSONLExporter{}
	}
	return &FileJSONLExporter{
		path: filepath.Join(projectRoot, "shark-data", ".stats", "events.jsonl"),
	}
}

// spanRecord is the JSON shape written for each span.
type spanRecord struct {
	Ts         string                 `json:"ts"`
	SpanName   string                 `json:"span_name"`
	TraceID    string                 `json:"trace_id"`
	SpanID     string                 `json:"span_id"`
	DurationMs int64                  `json:"duration_ms"`
	ExitStatus string                 `json:"exit_status"`
	Attrs      map[string]interface{} `json:"attrs"`
}

// ExportSpans appends one JSON line per span to the events.jsonl file.
// Errors are logged at slog.Debug level and swallowed — they are never returned.
func (e *FileJSONLExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	if e == nil || e.path == "" {
		return nil
	}
	if len(spans) == 0 {
		return nil
	}

	// Ensure the parent directory exists (fail-soft on error).
	dir := filepath.Dir(e.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Debug("file_jsonl_exporter: failed to create stats dir", "dir", dir, "err", err)
		return nil
	}

	// Open for append (create if missing).
	f, err := os.OpenFile(e.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Debug("file_jsonl_exporter: failed to open events file", "path", e.path, "err", err)
		return nil
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			slog.Debug("file_jsonl_exporter: failed to close events file", "path", e.path, "err", cerr)
		}
	}()

	enc := json.NewEncoder(f)
	for _, span := range spans {
		rec := toRecord(span)
		if err := enc.Encode(rec); err != nil {
			slog.Debug("file_jsonl_exporter: failed to encode span", "span_name", span.Name(), "err", err)
			// continue to next span
		}
	}
	return nil
}

// Shutdown is a no-op; there are no resources to release.
func (e *FileJSONLExporter) Shutdown(_ context.Context) error {
	return nil
}

// toRecord converts a ReadOnlySpan into the JSON-serialisable spanRecord shape.
func toRecord(span sdktrace.ReadOnlySpan) spanRecord {
	exitStatus := "ok"
	if span.Status().Code == codes.Error {
		exitStatus = "error"
	}

	attrs := make(map[string]interface{}, len(span.Attributes()))
	for _, kv := range span.Attributes() {
		attrs[string(kv.Key)] = attrValue(kv.Value)
	}

	return spanRecord{
		Ts:         span.EndTime().UTC().Format(time.RFC3339Nano),
		SpanName:   span.Name(),
		TraceID:    span.SpanContext().TraceID().String(),
		SpanID:     span.SpanContext().SpanID().String(),
		DurationMs: span.EndTime().Sub(span.StartTime()).Milliseconds(),
		ExitStatus: exitStatus,
		Attrs:      attrs,
	}
}

// attrValue extracts a Go-native value from an attribute.Value so it
// serialises cleanly as JSON (string, int64, float64, or bool).
func attrValue(v attribute.Value) interface{} {
	switch v.Type() {
	case attribute.BOOL:
		return v.AsBool()
	case attribute.INT64:
		return v.AsInt64()
	case attribute.FLOAT64:
		return v.AsFloat64()
	default:
		return v.AsString()
	}
}
