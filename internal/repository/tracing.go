package repository

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// repoTracer is the package-level OpenTelemetry tracer for all repository operations.
// When observability is disabled (the default), otel.Tracer() returns the global no-op
// tracer, making every span call a sub-microsecond no-op.
var repoTracer trace.Tracer = otel.Tracer("internal/repository")

// recordSpanError records an error on the span and sets its status to Error.
// This is a no-op if err is nil or span is nil.
func recordSpanError(span trace.Span, err error) {
	if err != nil && span != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}
