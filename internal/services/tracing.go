package services

import (
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// recordSpanError records an error on the span and sets the span status to Error.
// Returns the original error unchanged for use in return statements:
//
//	return nil, recordSpanError(span, err)
func recordSpanError(span trace.Span, err error) error {
	if err != nil && span != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}
