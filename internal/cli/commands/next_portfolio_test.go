package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type stubNextPortfolioAdvisor struct {
	advice *models.PortfolioAdviceEnvelope
	err    error
	calls  int
}

func (s *stubNextPortfolioAdvisor) Advise(context.Context) (*models.PortfolioAdviceEnvelope, error) {
	s.calls++
	return s.advice, s.err
}

func TestRunNextBareRoutesBeforeKeyedAdapters(t *testing.T) {
	originalAdvisorFactory := nextGetPortfolioAdvisor
	originalAdapterFactory := nextNewAdapterCache
	defer func() {
		nextGetPortfolioAdvisor = originalAdvisorFactory
		nextNewAdapterCache = originalAdapterFactory
	}()

	advisor := &stubNextPortfolioAdvisor{advice: models.NewPortfolioAdviceEnvelope()}
	nextGetPortfolioAdvisor = func() portfolioAdvisor { return advisor }
	nextNewAdapterCache = func(context.Context) (*nextAdapterCache, error) {
		t.Fatal("bare next constructed keyed adapters")
		return nil, nil
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(&bytes.Buffer{})
	if err := runNext(cmd, []string{}); err != nil {
		t.Fatalf("runNext() error = %v", err)
	}
	if advisor.calls != 1 {
		t.Fatalf("advisor calls = %d, want 1", advisor.calls)
	}
}

func TestNextCommandBareEmitsAdviceAndBoundedTelemetry(t *testing.T) {
	originalAdvisorFactory := nextGetPortfolioAdvisor
	originalAdapterFactory := nextNewAdapterCache
	originalTracerProvider := otel.GetTracerProvider()
	exporter := tracetest.NewInMemoryExporter()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tracerProvider)
	defer func() {
		nextGetPortfolioAdvisor = originalAdvisorFactory
		nextNewAdapterCache = originalAdapterFactory
		_ = tracerProvider.Shutdown(context.Background())
		otel.SetTracerProvider(originalTracerProvider)
	}()

	advice := models.NewPortfolioAdviceEnvelope()
	advice.EvidenceComplete = false
	advice.Prompt = "secret prompt text"
	advice.Epics = append(advice.Epics, models.PortfolioEpicEvidence{
		Key: "E01", EligibilityReasons: []string{}, BlockedItems: []models.PortfolioBlockedItem{},
		ActiveWork: []models.PortfolioActiveWork{{EntityType: "epic", EntityKey: "E01", ClaimedBy: "secret-holder"}},
	})
	advice.Relationships = append(advice.Relationships, models.PortfolioEpicRelationship{FromKey: "E01", ToKey: "E02"})
	advice.Ordering.Warnings = append(advice.Ordering.Warnings,
		models.PortfolioWarning{Code: models.PortfolioWarningHardOrderCycle, EpicKeys: []string{"E01"}},
		models.PortfolioWarning{Code: models.PortfolioWarningMissingOrdering, EpicKeys: []string{"E01", "E02"}},
	)
	advisor := &stubNextPortfolioAdvisor{advice: advice}
	nextGetPortfolioAdvisor = func() portfolioAdvisor { return advisor }
	nextNewAdapterCache = func(context.Context) (*nextAdapterCache, error) {
		t.Fatal("bare next constructed keyed adapters")
		return nil, nil
	}

	var stdout, stderr bytes.Buffer
	cmd := newNextCommand()
	cmd.SilenceUsage = true
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v; stderr = %s", err, stderr.String())
	}

	var decoded models.PortfolioAdviceEnvelope
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout is not one portfolio JSON object: %v\n%s", err, stdout.String())
	}
	if decoded.Mode != models.PortfolioAdviceModePortfolioAdvice || len(decoded.Epics) != 1 {
		t.Fatalf("decoded advice = %#v", decoded)
	}
	if advisor.calls != 1 {
		t.Fatalf("advisor calls = %d, want 1", advisor.calls)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 || spans[0].Name != "shark.next" {
		t.Fatalf("spans = %#v, want one shark.next span", spans)
	}
	wantAttributes := map[string]any{
		"mode":                          "portfolio_advice",
		"portfolio.candidate_count":     int64(1),
		"portfolio.relationship_count":  int64(1),
		"portfolio.graph_warning_count": int64(2),
		"portfolio.evidence_complete":   false,
	}
	gotAttributes := make(map[string]any, len(spans[0].Attributes))
	keys := make([]string, 0, len(spans[0].Attributes))
	for _, attr := range spans[0].Attributes {
		key := string(attr.Key)
		keys = append(keys, key)
		gotAttributes[key] = attr.Value.AsInterface()
		serialized := fmt.Sprint(attr.Value.AsInterface())
		if serialized == advice.Prompt || serialized == "secret-holder" || serialized == "E01" {
			t.Errorf("telemetry leaked portfolio payload through %s=%q", key, serialized)
		}
	}
	sort.Strings(keys)
	wantKeys := make([]string, 0, len(wantAttributes))
	for key := range wantAttributes {
		wantKeys = append(wantKeys, key)
	}
	sort.Strings(wantKeys)
	if !reflect.DeepEqual(keys, wantKeys) {
		t.Errorf("span attribute keys = %#v, want exactly %#v", keys, wantKeys)
	}
	if !reflect.DeepEqual(gotAttributes, wantAttributes) {
		t.Errorf("span attributes = %#v, want %#v", gotAttributes, wantAttributes)
	}
}

func TestNextCommandBareFatalErrorWritesNoJSONAndMarksSpan(t *testing.T) {
	originalAdvisorFactory := nextGetPortfolioAdvisor
	originalAdapterFactory := nextNewAdapterCache
	originalTracerProvider := otel.GetTracerProvider()
	exporter := tracetest.NewInMemoryExporter()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	otel.SetTracerProvider(tracerProvider)
	defer func() {
		nextGetPortfolioAdvisor = originalAdvisorFactory
		nextNewAdapterCache = originalAdapterFactory
		_ = tracerProvider.Shutdown(context.Background())
		otel.SetTracerProvider(originalTracerProvider)
	}()

	sentinel := errors.New("epic list unavailable")
	advisor := &stubNextPortfolioAdvisor{err: sentinel}
	nextGetPortfolioAdvisor = func() portfolioAdvisor { return advisor }
	nextNewAdapterCache = func(context.Context) (*nextAdapterCache, error) {
		t.Fatal("bare next constructed keyed adapters")
		return nil, nil
	}

	var stdout bytes.Buffer
	cmd := newNextCommand()
	cmd.SilenceUsage = true
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})
	err := cmd.ExecuteContext(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("ExecuteContext() error = %v, want wrapped sentinel", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want no partial JSON", stdout.String())
	}
	spans := exporter.GetSpans()
	if len(spans) != 1 || spans[0].Status.Code != codes.Error {
		t.Fatalf("spans = %#v, want one error-status span", spans)
	}
}

func TestNextCommandArgumentValidationRunsBeforeFactories(t *testing.T) {
	originalAdvisorFactory := nextGetPortfolioAdvisor
	originalAdapterFactory := nextNewAdapterCache
	defer func() {
		nextGetPortfolioAdvisor = originalAdvisorFactory
		nextNewAdapterCache = originalAdapterFactory
	}()

	advisorCalls, adapterCalls := 0, 0
	nextGetPortfolioAdvisor = func() portfolioAdvisor {
		advisorCalls++
		t.Fatal("invalid arguments constructed portfolio advisor")
		return nil
	}
	nextNewAdapterCache = func(context.Context) (*nextAdapterCache, error) {
		adapterCalls++
		t.Fatal("invalid arguments constructed keyed adapters")
		return nil, nil
	}

	for _, args := range [][]string{{"E36", "extra"}, {"E36", "extra", "third"}} {
		cmd := newNextCommand()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(args)
		err := cmd.ExecuteContext(context.Background())
		if err == nil || !strings.Contains(err.Error(), "accepts at most 1 arg(s)") {
			t.Errorf("args %v error = %v, want Cobra maximum-arguments error", args, err)
		}
	}
	if advisorCalls != 0 || adapterCalls != 0 {
		t.Fatalf("factory calls = advisor:%d adapter:%d, want zero", advisorCalls, adapterCalls)
	}
}

func TestNextCommandPreviewIsUnknownBeforeFactories(t *testing.T) {
	originalAdvisorFactory := nextGetPortfolioAdvisor
	originalAdapterFactory := nextNewAdapterCache
	defer func() {
		nextGetPortfolioAdvisor = originalAdvisorFactory
		nextNewAdapterCache = originalAdapterFactory
	}()

	advisorCalls, adapterCalls := 0, 0
	nextGetPortfolioAdvisor = func() portfolioAdvisor {
		advisorCalls++
		t.Fatal("unknown flag constructed portfolio advisor")
		return nil
	}
	nextNewAdapterCache = func(context.Context) (*nextAdapterCache, error) {
		adapterCalls++
		t.Fatal("unknown flag constructed keyed adapters")
		return nil, nil
	}

	for _, args := range [][]string{{"--preview"}, {"E36", "--preview"}} {
		cmd := newNextCommand()
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(args)
		err := cmd.ExecuteContext(context.Background())
		if err == nil || !strings.Contains(err.Error(), "unknown flag: --preview") {
			t.Errorf("args %v error = %v, want Cobra unknown-flag error", args, err)
		}
	}
	if advisorCalls != 0 || adapterCalls != 0 {
		t.Fatalf("factory calls = advisor:%d adapter:%d, want zero", advisorCalls, adapterCalls)
	}
	if nextCmd.Flags().Lookup("preview") != nil {
		t.Error("next command unexpectedly exposes preview")
	}
	if taskNextStatusCmd.Flags().Lookup("preview") == nil {
		t.Error("task lifecycle preview flag was removed")
	}
}

func TestNextCommandKeyedDoesNotConstructPortfolioAdvisor(t *testing.T) {
	originalAdvisorFactory := nextGetPortfolioAdvisor
	originalAdapterFactory := nextNewAdapterCache
	defer func() {
		nextGetPortfolioAdvisor = originalAdvisorFactory
		nextNewAdapterCache = originalAdapterFactory
	}()

	advisorCalls, adapterCalls := 0, 0
	nextGetPortfolioAdvisor = func() portfolioAdvisor {
		advisorCalls++
		t.Fatal("keyed next constructed portfolio advisor")
		return nil
	}
	sentinel := errors.New("keyed adapter sentinel")
	nextNewAdapterCache = func(context.Context) (*nextAdapterCache, error) {
		adapterCalls++
		return nil, sentinel
	}

	cmd := newNextCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"E36"})
	err := cmd.ExecuteContext(context.Background())
	if !errors.Is(err, sentinel) {
		t.Fatalf("ExecuteContext() error = %v, want keyed adapter sentinel", err)
	}
	if advisorCalls != 0 || adapterCalls != 1 {
		t.Fatalf("factory calls = advisor:%d adapter:%d, want 0 and 1", advisorCalls, adapterCalls)
	}
}
