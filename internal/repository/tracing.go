package repository

import (
	"go.opentelemetry.io/otel/trace"

	"github.com/jwwelbor/shark-task-manager/internal/repository/repoutil"
)

// repoTracer is the package-level OpenTelemetry tracer for the root repository package.
// Used by tracing_test.go to validate span creation. Sub-packages (epic, feature, task)
// each maintain their own package-level tracer via repoutil.NewTracer.
var repoTracer trace.Tracer = repoutil.NewTracer("internal/repository")
