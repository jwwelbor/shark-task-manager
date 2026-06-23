package observability

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// makeTestSpans returns two synthetic ReadOnlySpan values using the SDK's
// SpanStub helpers so no real OTel provider needs to be initialised.
func makeTestSpans(t *testing.T) []sdktrace.ReadOnlySpan {
	t.Helper()

	now := time.Now()

	stubs := tracetest.SpanStubs{
		{
			Name: "shark.next",
			SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				SpanID:     [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
				TraceFlags: trace.FlagsSampled,
			}),
			StartTime: now,
			EndTime:   now.Add(47 * time.Millisecond),
			Attributes: []attribute.KeyValue{
				attribute.String("command", "task next"),
				attribute.Int64("exit_code", 0),
			},
		},
		{
			Name: "shark.status",
			SpanContext: trace.NewSpanContext(trace.SpanContextConfig{
				TraceID:    [16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
				SpanID:     [8]byte{8, 7, 6, 5, 4, 3, 2, 1},
				TraceFlags: trace.FlagsSampled,
			}),
			StartTime: now.Add(100 * time.Millisecond),
			EndTime:   now.Add(123 * time.Millisecond),
			Attributes: []attribute.KeyValue{
				attribute.String("command", "status"),
				attribute.Bool("json_output", true),
			},
		},
	}

	return stubs.Snapshots()
}

func TestFileJSONLExporter_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	// The exporter takes projectRoot; it appends shark-data/.stats/events.jsonl
	exporter := NewFileJSONLExporter(tmpDir)

	spans := makeTestSpans(t)
	require.Len(t, spans, 2)

	err := exporter.ExportSpans(context.Background(), spans)
	require.NoError(t, err)

	// Read and parse the JSONL file.
	outPath := filepath.Join(tmpDir, "shark-data", ".stats", "events.jsonl")
	f, err := os.Open(outPath)
	require.NoError(t, err, "events.jsonl should have been created")
	defer f.Close()

	var lines []map[string]interface{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		assert.NotEmpty(t, line, "each line must be non-empty")
		var rec map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(line), &rec), "each line must be valid JSON")
		lines = append(lines, rec)
	}
	require.NoError(t, scanner.Err())

	// Assert exactly 2 lines were written.
	require.Len(t, lines, 2, "one JSON line per span")

	// Assert required fields are present in both records.
	for i, rec := range lines {
		assert.NotEmpty(t, rec["ts"], "line %d: ts must be present", i)
		assert.NotEmpty(t, rec["span_name"], "line %d: span_name must be present", i)
		assert.NotEmpty(t, rec["trace_id"], "line %d: trace_id must be present", i)
		assert.NotEmpty(t, rec["span_id"], "line %d: span_id must be present", i)
		assert.NotNil(t, rec["duration_ms"], "line %d: duration_ms must be present", i)
		assert.NotNil(t, rec["attrs"], "line %d: attrs must be present", i)
		assert.NotEmpty(t, rec["exit_status"], "line %d: exit_status must be present", i)
	}

	// First span: command attr should survive round-trip.
	line0Attrs, ok := lines[0]["attrs"].(map[string]interface{})
	require.True(t, ok, "attrs must be a JSON object")
	assert.Equal(t, "task next", line0Attrs["command"])

	// Second span: bool attr should survive round-trip.
	line1Attrs, ok := lines[1]["attrs"].(map[string]interface{})
	require.True(t, ok, "attrs must be a JSON object")
	assert.Equal(t, true, line1Attrs["json_output"])
}

func TestFileJSONLExporter_CreatesStatsDir(t *testing.T) {
	tmpDir := t.TempDir()
	exporter := NewFileJSONLExporter(tmpDir)

	spans := makeTestSpans(t)
	err := exporter.ExportSpans(context.Background(), spans)
	require.NoError(t, err)

	// The shark-data/.stats directory must have been created automatically.
	statsDir := filepath.Join(tmpDir, "shark-data", ".stats")
	info, err := os.Stat(statsDir)
	require.NoError(t, err, "stats dir should exist")
	assert.True(t, info.IsDir(), "stats path should be a directory")
}

func TestFileJSONLExporter_EmptyRootSkipsWrites(t *testing.T) {
	// NewFileJSONLExporter("") must not panic or return an error.
	exporter := NewFileJSONLExporter("")

	spans := makeTestSpans(t)
	err := exporter.ExportSpans(context.Background(), spans)
	assert.NoError(t, err, "empty root: ExportSpans must not error")
}

func TestFileJSONLExporter_NilSafe(t *testing.T) {
	var exporter *FileJSONLExporter

	spans := makeTestSpans(t)
	err := exporter.ExportSpans(context.Background(), spans)
	assert.NoError(t, err, "nil exporter: ExportSpans must not panic or error")
}

func TestFileJSONLExporter_ShutdownReturnsNil(t *testing.T) {
	tmpDir := t.TempDir()
	exporter := NewFileJSONLExporter(tmpDir)
	assert.NoError(t, exporter.Shutdown(context.Background()))
}

func TestFileJSONLExporter_EmptySpanSlice(t *testing.T) {
	tmpDir := t.TempDir()
	exporter := NewFileJSONLExporter(tmpDir)

	err := exporter.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{})
	require.NoError(t, err)

	// No file should be created when there are no spans.
	outPath := filepath.Join(tmpDir, "shark-data", ".stats", "events.jsonl")
	_, err = os.Stat(outPath)
	assert.True(t, os.IsNotExist(err), "events.jsonl should not be created for empty spans")
}

func TestFileJSONLExporter_ImplementsSpanExporter(t *testing.T) {
	// Compile-time interface check surfaced as a runtime assertion.
	var _ sdktrace.SpanExporter = (*FileJSONLExporter)(nil)
}
