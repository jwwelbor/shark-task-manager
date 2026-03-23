package repoutil

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// NewTracer creates an OTel tracer scoped to a repository sub-package.
// The name parameter should be the fully-qualified sub-package path
// (e.g., "internal/repository/task") to enable per-package span attribution.
//
// When observability is disabled (the default), otel.Tracer() returns the
// global no-op tracer, making every span call a sub-microsecond no-op.
func NewTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// RecordSpanError records an error on the span and sets its status to Error.
// This is a no-op if err is nil or span is nil.
func RecordSpanError(span trace.Span, err error) {
	if err != nil && span != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}
