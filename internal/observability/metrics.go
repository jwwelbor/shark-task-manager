package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// CommandMetrics holds OTel metric instruments for CLI command tracking.
// Obtain via NewCommandMetrics(). Zero value is safe to use (all methods are no-ops
// because nil instrument checks are performed before recording).
type CommandMetrics struct {
	duration    metric.Float64Histogram
	invocations metric.Int64Counter
	errors      metric.Int64Counter
}

// NewCommandMetrics creates CLI command metric instruments against the given meter.
// Returns an error if any instrument registration fails.
func NewCommandMetrics(meter metric.Meter) (CommandMetrics, error) {
	duration, err := meter.Float64Histogram(
		"shark.cli.command.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Duration of CLI command execution in milliseconds"),
	)
	if err != nil {
		return CommandMetrics{}, err
	}

	invocations, err := meter.Int64Counter(
		"shark.cli.command.invocations",
		metric.WithDescription("Total number of CLI command invocations"),
	)
	if err != nil {
		return CommandMetrics{}, err
	}

	errs, err := meter.Int64Counter(
		"shark.cli.command.errors",
		metric.WithDescription("Total number of CLI command errors"),
	)
	if err != nil {
		return CommandMetrics{}, err
	}

	return CommandMetrics{
		duration:    duration,
		invocations: invocations,
		errors:      errs,
	}, nil
}

// statusFromErr returns "ok" if err is nil, "error" otherwise.
func statusFromErr(err error) string {
	if err != nil {
		return "error"
	}
	return "ok"
}

// RecordDuration records the execution duration of a CLI command.
// command is the cobra command use string (e.g., "task get").
// err is nil on success; non-nil sets the status attribute to "error".
func (m CommandMetrics) RecordDuration(ctx context.Context, command string, duration time.Duration, err error) {
	if m.duration == nil {
		return
	}
	m.duration.Record(ctx, float64(duration.Milliseconds()),
		metric.WithAttributes(
			attribute.String("command", command),
			attribute.String("status", statusFromErr(err)),
		),
	)
}

// RecordInvocation increments the command invocation counter.
// err determines the status attribute value ("ok" or "error").
func (m CommandMetrics) RecordInvocation(ctx context.Context, command string, err error) {
	if m.invocations == nil {
		return
	}
	m.invocations.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("command", command),
			attribute.String("status", statusFromErr(err)),
		),
	)

	// Also record error count if applicable
	if err != nil && m.errors != nil {
		m.errors.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("command", command),
				attribute.String("error_type", errorType(err)),
			),
		)
	}
}

// DBMetrics holds OTel metric instruments for database query tracking.
// Obtain via NewDBMetrics(). Zero value is safe to use (all methods are no-ops).
type DBMetrics struct {
	queryDuration metric.Float64Histogram
	queryErrors   metric.Int64Counter
}

// NewDBMetrics creates database query metric instruments against the given meter.
// Returns an error if any instrument registration fails.
func NewDBMetrics(meter metric.Meter) (DBMetrics, error) {
	queryDuration, err := meter.Float64Histogram(
		"shark.db.query.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Duration of database queries in milliseconds"),
	)
	if err != nil {
		return DBMetrics{}, err
	}

	queryErrors, err := meter.Int64Counter(
		"shark.db.query.errors",
		metric.WithDescription("Total number of database query errors"),
	)
	if err != nil {
		return DBMetrics{}, err
	}

	return DBMetrics{
		queryDuration: queryDuration,
		queryErrors:   queryErrors,
	}, nil
}

// RecordQueryDuration records the execution duration of a database query.
// operation is the SQL operation type (e.g., "SELECT", "INSERT").
// table is the database table name.
// err is nil on success; non-nil records an error count.
func (m DBMetrics) RecordQueryDuration(ctx context.Context, operation, table string, duration time.Duration, err error) {
	if m.queryDuration == nil {
		return
	}
	attrs := metric.WithAttributes(
		attribute.String("operation", operation),
		attribute.String("table", table),
	)
	m.queryDuration.Record(ctx, float64(duration.Milliseconds()), attrs)

	if err != nil && m.queryErrors != nil {
		m.queryErrors.Add(ctx, 1, attrs)
	}
}

// errorType extracts a short error type string from an error for metric attributes.
func errorType(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if len(msg) > 50 {
		msg = msg[:50]
	}
	return msg
}
