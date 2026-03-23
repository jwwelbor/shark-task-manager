package note

import "go.opentelemetry.io/otel/trace"

// SetTracerForTesting replaces the package-level noteTracer with the provided tracer.
// It returns a restore function that reverts to the original tracer.
// This is intended for use in test setup/teardown only.
func SetTracerForTesting(t trace.Tracer) (restore func()) {
	prev := noteTracer
	noteTracer = t
	return func() { noteTracer = prev }
}
