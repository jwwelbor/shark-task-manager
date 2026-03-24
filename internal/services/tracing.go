package services

import (
	"go.opentelemetry.io/otel/trace"

	"github.com/jwwelbor/shark-task-manager/internal/repository/repoutil"
)

// recordSpanError records an error on the span and sets the span status to Error.
// Returns the original error unchanged for use in return statements:
//
//	return nil, recordSpanError(span, err)
//
// Delegates to repoutil.RecordSpanError, which is the canonical implementation.
func recordSpanError(span trace.Span, err error) error {
	repoutil.RecordSpanError(span, err)
	return err
}
