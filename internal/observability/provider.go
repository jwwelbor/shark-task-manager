package observability

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/jwwelbor/shark-task-manager/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
)

const (
	defaultServiceName = "shark-task-manager"
	defaultExporter    = "stdout"
	defaultOTLPProto   = "grpc"
	serviceVersion     = "0.1.0"
)

// InitProvider initializes the OpenTelemetry TracerProvider and MeterProvider.
// When cfg.Enabled is false, delegates to NoopProvider() -- no SDK is initialized,
// no network connections are made, and the returned ShutdownFunc is a no-op.
// Must be called once during application startup (before any span/metric creation).
// Returns a ShutdownFunc that must be deferred or called before process exit.
//
// For callers that know the project root (e.g. the CLI), use InitProviderWithRoot
// so that the "file_jsonl" exporter can resolve the output path correctly.
func InitProvider(cfg config.ObservabilityConfig) (ShutdownFunc, error) {
	return InitProviderWithRoot(cfg, "")
}

// InitProviderWithRoot is like InitProvider but accepts a projectRoot string that
// is forwarded to the "file_jsonl" exporter so it can resolve its output path
// without importing the cli package (which would create a circular dependency).
// When projectRoot is empty, the file_jsonl exporter silently skips writes.
func InitProviderWithRoot(cfg config.ObservabilityConfig, projectRoot string) (ShutdownFunc, error) {
	if !cfg.Enabled {
		return NoopProvider(), nil
	}

	res, err := buildResource(cfg)
	if err != nil {
		return NoopProvider(), fmt.Errorf("failed to create OTel resource: %w", err)
	}

	var tp *sdktrace.TracerProvider
	if cfg.TracingEnabled {
		tp, err = buildTracerProvider(cfg, res, projectRoot)
		if err != nil {
			return NoopProvider(), fmt.Errorf("failed to create tracer provider: %w", err)
		}
	}

	var mp *sdkmetric.MeterProvider
	if cfg.MetricsEnabled {
		mp, err = buildMeterProvider(cfg, res)
		if err != nil {
			// Clean up tracer provider if meter provider fails
			if tp != nil {
				_ = tp.Shutdown(context.Background())
			}
			return NoopProvider(), fmt.Errorf("failed to create meter provider: %w", err)
		}
	}

	// Set global providers
	if tp != nil {
		otel.SetTracerProvider(tp)
	}
	if mp != nil {
		otel.SetMeterProvider(mp)
	}

	// Set W3C trace context propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Build idempotent shutdown function
	var shutdownOnce sync.Once
	shutdown := func(ctx context.Context) error {
		var errs []error
		shutdownOnce.Do(func() {
			if tp != nil {
				if err := tp.Shutdown(ctx); err != nil {
					errs = append(errs, fmt.Errorf("tracer provider shutdown: %w", err))
				}
			}
			if mp != nil {
				if err := mp.Shutdown(ctx); err != nil {
					errs = append(errs, fmt.Errorf("meter provider shutdown: %w", err))
				}
			}
		})
		return errors.Join(errs...)
	}

	return shutdown, nil
}

// ApplyEnvOverrides applies environment variable overrides on top of the config struct values.
// Should be called once before passing cfg to InitLogger and InitProvider.
func ApplyEnvOverrides(cfg *config.ObservabilityConfig) {
	if v := os.Getenv("SHARK_OTEL_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Enabled = b
		}
	}
	if v := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); v != "" {
		cfg.OTLPEndpoint = v
	}
	if v := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"); v != "" {
		cfg.OTLPProtocol = v
	}
	if v := os.Getenv("OTEL_SERVICE_NAME"); v != "" {
		cfg.ServiceName = v
	}
	if v := os.Getenv("SHARK_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("SHARK_LOG_FORMAT"); v != "" {
		cfg.LogFormat = v
	}
	if v := os.Getenv("SHARK_LOG_FILE"); v != "" {
		cfg.LogFile = v
	}
}

func buildResource(cfg config.ObservabilityConfig) (*resource.Resource, error) {
	svcName := cfg.ServiceName
	if svcName == "" {
		svcName = defaultServiceName
	}
	return resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(svcName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
}

// resolveExporter normalizes the exporter name, defaulting to "stdout".
func resolveExporter(exporter string) string {
	exporter = strings.ToLower(strings.TrimSpace(exporter))
	if exporter == "" {
		return defaultExporter
	}
	return exporter
}

// resolveOTLPEndpoint returns the OTLP endpoint, defaulting to localhost:4317.
func resolveOTLPEndpoint(endpoint string) string {
	if endpoint == "" {
		return "localhost:4317"
	}
	return endpoint
}

func buildTracerProvider(cfg config.ObservabilityConfig, res *resource.Resource, projectRoot string) (*sdktrace.TracerProvider, error) {
	exporter := resolveExporter(cfg.Exporter)

	var opts []sdktrace.TracerProviderOption
	opts = append(opts, sdktrace.WithResource(res))

	// Configure sample rate
	if cfg.SampleRate > 0 && cfg.SampleRate < 1.0 {
		opts = append(opts, sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)))
	} else {
		opts = append(opts, sdktrace.WithSampler(sdktrace.AlwaysSample()))
	}

	switch exporter {
	case "stdout":
		// Write to os.Stderr, never stdout
		exp, err := stdouttrace.New(stdouttrace.WithWriter(os.Stderr))
		if err != nil {
			return nil, fmt.Errorf("stdout trace exporter: %w", err)
		}
		// Use SimpleSpanProcessor for stdout (ADR-6: immediate output for dev)
		opts = append(opts, sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)))

	case "otlp":
		endpoint := resolveOTLPEndpoint(cfg.OTLPEndpoint)
		exp, err := otlptracegrpc.New(
			context.Background(),
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithInsecure(), // Allow non-TLS for local dev; production should use TLS
		)
		if err != nil {
			return nil, fmt.Errorf("otlp trace exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(exp))

	case "file_jsonl":
		// Append one JSON line per span to <projectRoot>/shark-data/.stats/events.jsonl.
		// Uses SimpleSpanProcessor (immediate flush, no batching) — correct for low-volume
		// per-CLI-call telemetry where batching would defer writes past process exit.
		// When projectRoot is empty, the exporter silently skips writes (fail-soft).
		exp := NewFileJSONLExporter(projectRoot)
		opts = append(opts, sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exp)))

	default:
		return nil, fmt.Errorf("unsupported exporter type: %q (expected \"stdout\", \"otlp\", or \"file_jsonl\")", exporter)
	}

	return sdktrace.NewTracerProvider(opts...), nil
}

func buildMeterProvider(cfg config.ObservabilityConfig, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	exporter := resolveExporter(cfg.Exporter)

	var opts []sdkmetric.Option
	opts = append(opts, sdkmetric.WithResource(res))

	switch exporter {
	case "stdout":
		// Write to os.Stderr, never stdout
		exp, err := stdoutmetric.New(stdoutmetric.WithWriter(os.Stderr))
		if err != nil {
			return nil, fmt.Errorf("stdout metric exporter: %w", err)
		}
		opts = append(opts, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)))

	case "otlp":
		endpoint := resolveOTLPEndpoint(cfg.OTLPEndpoint)
		exp, err := otlpmetricgrpc.New(
			context.Background(),
			otlpmetricgrpc.WithEndpoint(endpoint),
			otlpmetricgrpc.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("otlp metric exporter: %w", err)
		}
		opts = append(opts, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp)))

	default:
		return nil, fmt.Errorf("unsupported exporter type: %q (expected \"stdout\" or \"otlp\")", exporter)
	}

	return sdkmetric.NewMeterProvider(opts...), nil
}
