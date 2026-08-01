package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type questionListServiceStub struct {
	listed            bool
	updated           bool
	transitioned      bool
	configured        bool
	responded         bool
	resolved          bool
	withdrawn         bool
	superseded        bool
	configureIn       *services.ConfigureWorkflowInput
	responseIn        *services.RecordQuestionResponseInput
	resolveIn         *services.ResolveQuestionInput
	withdrawIn        *services.WithdrawQuestionInput
	supersedeIn       *services.SupersedeQuestionInput
	question          *models.Question
	openByResponderIn *openByResponderInput
	blockingForIn     *blockingForInput
	fullReadIn        *fullReadInput
}

type openByResponderInput struct {
	responder     string
	limit, offset int
}
type blockingForInput struct {
	targetType    models.EntityType
	targetKey     string
	limit, offset int
}
type fullReadInput struct{ key, actor string }

func (s *questionListServiceStub) workflowQuestion() *models.Question {
	if s.question != nil {
		return s.question
	}
	return &models.Question{BaseEntity: models.BaseEntity{Key: "Q001", Title: "Question"}}
}

func (s *questionListServiceStub) CreateQuestion(context.Context, services.CreateQuestionInput) (*models.Question, error) {
	return nil, nil
}
func (s *questionListServiceStub) GetQuestion(context.Context, string) (*models.Question, error) {
	return nil, nil
}
func (s *questionListServiceStub) ListQuestions(context.Context, services.QuestionListFilter) ([]*models.Question, error) {
	s.listed = true
	return nil, nil
}
func (s *questionListServiceStub) UpdateQuestion(context.Context, string, services.QuestionUpdates) (*models.Question, error) {
	s.updated = true
	return &models.Question{BaseEntity: models.BaseEntity{Key: "Q001", Title: "Updated question"}}, nil
}
func (s *questionListServiceStub) DeleteQuestion(context.Context, string) error { return nil }
func (s *questionListServiceStub) GetNextStatus(_ context.Context, key string) (*services.NextStatusInfo, error) {
	return &services.NextStatusInfo{EntityType: models.EntityTypeQuestion, EntityKey: key, CurrentStatus: "draft"}, nil
}
func (s *questionListServiceStub) TransitionStatus(_ context.Context, key, target string, _ services.TransitionOptions) (*services.TransitionResult, error) {
	s.transitioned = true
	return &services.TransitionResult{EntityType: models.EntityTypeQuestion, EntityKey: key, FromStatus: "draft", ToStatus: target, Transitioned: true}, nil
}
func (s *questionListServiceStub) ConfigureWorkflow(_ context.Context, in services.ConfigureWorkflowInput) (*models.Question, error) {
	s.configured = true
	s.configureIn = &in
	return s.workflowQuestion(), nil
}
func (s *questionListServiceStub) RecordResponse(_ context.Context, in services.RecordQuestionResponseInput) (*models.Question, error) {
	s.responded = true
	s.responseIn = &in
	return s.workflowQuestion(), nil
}
func (s *questionListServiceStub) Resolve(_ context.Context, in services.ResolveQuestionInput) (*models.Question, error) {
	s.resolved = true
	s.resolveIn = &in
	return s.workflowQuestion(), nil
}
func (s *questionListServiceStub) Withdraw(_ context.Context, in services.WithdrawQuestionInput) (*models.Question, error) {
	s.withdrawn = true
	s.withdrawIn = &in
	return s.workflowQuestion(), nil
}
func (s *questionListServiceStub) Supersede(_ context.Context, in services.SupersedeQuestionInput) (*models.Question, error) {
	s.superseded = true
	s.supersedeIn = &in
	return s.workflowQuestion(), nil
}
func (s *questionListServiceStub) ListOpenQuestionsByResponder(_ context.Context, responder string, limit, offset int) ([]models.QuestionProjection, error) {
	s.openByResponderIn = &openByResponderInput{responder, limit, offset}
	return []models.QuestionProjection{{Key: "Q001", Title: "Open", Status: "open"}}, nil
}
func (s *questionListServiceStub) ListQuestionsBlocking(_ context.Context, targetType models.EntityType, targetKey string, limit, offset int) ([]*services.QuestionBlock, error) {
	s.blockingForIn = &blockingForInput{targetType, targetKey, limit, offset}
	return []*services.QuestionBlock{{QuestionKey: "Q001", Summary: "Blocks", ResolutionOwner: "owner", CurrentResponder: "alice"}}, nil
}
func (s *questionListServiceStub) ReadQuestionFull(_ context.Context, key, actor string) (*models.QuestionFullProjection, error) {
	s.fullReadIn = &fullReadInput{key, actor}
	return &models.QuestionFullProjection{QuestionProjection: models.QuestionProjection{Key: key, Title: "Full", Status: "answering"}, ResolutionOwner: "owner", Responders: []models.QuestionResponder{{Identity: "alice"}}}, nil
}

var _ questionServicer = (*questionListServiceStub)(nil)

// TC-008: status commands auto-detect Q keys and dispatch through the Question
// service rather than rejecting Question as an unsupported generic entity type.
func TestQuestionStatusDispatchesThroughGenericCLI_TC008(t *testing.T) {
	stub := &questionListServiceStub{}
	withQuestionSvcOverride(t, stub)
	info, err := dispatchNextStatus(context.Background(), "question", "Q001")
	if err != nil || info.EntityKey != "Q001" || info.CurrentStatus != "draft" {
		t.Fatalf("dispatchNextStatus(question, Q001) = %#v, %v", info, err)
	}
	result, err := dispatchTransition(context.Background(), "question", "Q001", "archived", services.TransitionOptions{})
	if err != nil || !stub.transitioned || result.ToStatus != "archived" {
		t.Fatalf("dispatchTransition(question, Q001, archived) = %#v, %v; transitioned=%v", result, err, stub.transitioned)
	}
}

func withQuestionSvcOverride(t *testing.T, svc questionServicer) {
	t.Helper()
	previous := questionSvcOverride
	questionSvcOverride = svc
	t.Cleanup(func() { questionSvcOverride = previous })
}

func newQuestionListCommand(runE func(*cobra.Command, []string) error) *cobra.Command {
	cmd := &cobra.Command{Use: "list", RunE: runE}
	cmd.Flags().String("status", "", "")
	cmd.Flags().String("requester", "", "")
	cmd.Flags().String("blocking", "", "")
	cmd.Flags().Int("limit", 50, "")
	cmd.Flags().Int("offset", 0, "")
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return cmd
}

// `question list` owns only its finite filter set. Cobra rejects every other
// flag before the service can observe the list request.
func TestQuestionListCommandRejectsUndeclaredFilterFlags(t *testing.T) {
	for _, flag := range []string{"--sort-by=key", "--filter=owner", "--tag=release"} {
		t.Run(flag, func(t *testing.T) {
			stub := &questionListServiceStub{}
			withQuestionSvcOverride(t, stub)
			cmd := newQuestionListCommand(runQuestionList)
			cmd.SetArgs([]string{flag})
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "unknown flag") {
				t.Fatalf("Execute() error = %v, want unknown flag", err)
			}
			if stub.listed {
				t.Fatal("ListQuestions called for an undeclared flag")
			}
		})
	}
}

// `shark list question` shares list-level flags with other entity types. It
// must reject the flags that the Question query contract does not declare.
func TestUnifiedQuestionListRejectsUnsupportedSharedFilters(t *testing.T) {
	for _, flag := range []string{"--sort-by=key", "--show-all", "--all", "--priority=1", "--tag=release"} {
		t.Run(flag, func(t *testing.T) {
			stub := &questionListServiceStub{}
			withQuestionSvcOverride(t, stub)
			cmd := newQuestionListCommand(runList)
			cmd.Flags().String("sort-by", "", "")
			cmd.Flags().Bool("show-all", false, "")
			cmd.Flags().Bool("all", false, "")
			cmd.Flags().Int("priority", 0, "")
			cmd.Flags().StringSlice("tag", nil, "")
			cmd.SetArgs([]string{"question", flag})
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "unsupported Question list flag") {
				t.Fatalf("Execute() error = %v, want unsupported Question list flag", err)
			}
			if stub.listed {
				t.Fatal("ListQuestions called for an unsupported shared list flag")
			}
		})
	}
}

// The unified update command deliberately exposes flags for every entity
// type. Question accepts only its finite base-record fields, and must reject
// the others before the service receives an update request.
func TestUnifiedQuestionUpdateRejectsUnsupportedChangedFlagsBeforeService(t *testing.T) {
	for _, flag := range []string{"--key=Q010", "--file=docs/question.md", "--priority=1", "--tag=release", "--order=1", "--force"} {
		t.Run(flag, func(t *testing.T) {
			stub := &questionListServiceStub{}
			withQuestionSvcOverride(t, stub)
			cmd := buildIsolatedDispatchCmd(t)
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs([]string{"Q001", flag})
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "unsupported Question update flag") {
				t.Fatalf("Execute() error = %v, want unsupported Question update flag", err)
			}
			if stub.updated {
				t.Fatal("UpdateQuestion called for unsupported generic update flag")
			}
		})
	}
}

// newQuestionInvocationRoot builds the full Cobra parent/child relationship
// used by the CLI. The persistent flags are deliberately inherited rather
// than registered on the child so these tests protect the flag-scope boundary
// in Question validators.
func newQuestionInvocationRoot(t *testing.T, command *cobra.Command) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "shark"}
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.PersistentFlags().String("db", "", "database path")
	root.PersistentFlags().BoolVar(&cli.GlobalConfig.JSON, "json", false, "JSON output")
	root.PersistentFlags().Bool("no-color", false, "disable color")
	root.AddCommand(command)
	return root
}

// The unified update command must reject unsupported local mutation flags but
// accept inherited execution flags. This executes the full Cobra command tree
// rather than invoking runUpdate directly, so Flag.Visit sees the same merged
// flag view as a real `shark --db ... --json ... update Q001 ...` invocation.
func TestUnifiedQuestionUpdateAcceptsInheritedGlobalFlags(t *testing.T) {
	originalJSON := cli.GlobalConfig.JSON
	t.Cleanup(func() { cli.GlobalConfig.JSON = originalJSON })

	stub := &questionListServiceStub{}
	withQuestionSvcOverride(t, stub)
	command := buildIsolatedDispatchCmd(t)
	command.Use = "update <KEY> [flags]"
	root := newQuestionInvocationRoot(t, command)
	root.SetArgs([]string{
		"--db", "question-test.db",
		"--json",
		"--no-color",
		"update", "Q001", "--title", "Updated through inherited flags",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, want inherited global flags accepted", err)
	}
	if !stub.updated {
		t.Fatal("UpdateQuestion was not called for a supported Question update")
	}
}

// The direct list command has the same persistent-flag boundary as unified
// update. Its validator must continue to ignore inherited execution flags
// while enforcing the finite set of command-local Question filters.
func TestQuestionListAcceptsInheritedGlobalFlags(t *testing.T) {
	originalJSON := cli.GlobalConfig.JSON
	t.Cleanup(func() { cli.GlobalConfig.JSON = originalJSON })

	stub := &questionListServiceStub{}
	withQuestionSvcOverride(t, stub)
	command := newQuestionListCommand(runQuestionList)
	command.Use = "list"
	question := &cobra.Command{Use: "question"}
	question.AddCommand(command)
	root := newQuestionInvocationRoot(t, question)
	root.SetArgs([]string{
		"--db", "question-test.db",
		"--json",
		"--no-color",
		"question", "list",
	})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, want inherited global flags accepted", err)
	}
	if !stub.listed {
		t.Fatal("ListQuestions was not called with inherited global flags")
	}
}

// TC-401: the focused responder command passes only the declared identity and
// bounded page to the service, and emits the compact projection.
func TestQuestionOpenByResponderCommandTC401(t *testing.T) {
	originalJSON := cli.GlobalConfig.JSON
	t.Cleanup(func() { cli.GlobalConfig.JSON = originalJSON })
	cli.GlobalConfig.JSON = true
	stub := &questionListServiceStub{}
	withQuestionSvcOverride(t, stub)
	cmd := newQuestionOpenByResponderCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"alice", "--limit", "1", "--offset", "0"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("TC-401 command error = %v", err)
	}
	if got, want := *stub.openByResponderIn, (openByResponderInput{"alice", 1, 0}); got != want {
		t.Fatalf("TC-401 service input = %#v, want %#v", got, want)
	}
}

// TC-401: malformed pages and undeclared flags must stop at Cobra's command
// boundary, before a focused read can observe the request.
func TestQuestionOpenByResponderRejectsInvalidInputBeforeServiceTC401(t *testing.T) {
	for _, args := range [][]string{{"alice", "--limit", "0"}, {"alice", "--offset", "-1"}, {"alice", "--unknown", "x"}, {" alice "}, {string([]byte{0xff})}, {strings.Repeat("a", 257)}, {"bearer token"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			stub := &questionListServiceStub{}
			withQuestionSvcOverride(t, stub)
			cmd := newQuestionOpenByResponderCommand()
			cmd.SetArgs(args)
			if err := cmd.Execute(); err == nil {
				t.Fatal("TC-401 command error = nil, want validation rejection")
			}
			if stub.openByResponderIn != nil {
				t.Fatal("TC-401 focused service called for invalid input")
			}
		})
	}
}

// TC-403: the blocking command resolves a non-Question target from its key
// and forwards only the declared finite page to the shared F03 service seam.
func TestQuestionBlockingForCommandTC403(t *testing.T) {
	stub := &questionListServiceStub{}
	withQuestionSvcOverride(t, stub)
	cmd := newQuestionBlockingForCommand()
	cmd.SetArgs([]string{"E39-F04", "--limit", "50", "--offset", "0"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("TC-403 command error = %v", err)
	}
	if got, want := *stub.blockingForIn, (blockingForInput{models.EntityTypeFeature, "E39-F04", 50, 0}); got != want {
		t.Fatalf("TC-403 service input = %#v, want %#v", got, want)
	}
}

// TC-403: Question targets, malformed pages, and undeclared flags do not call
// the blocker read, preserving the direct-only service boundary.
func TestQuestionBlockingForRejectsInvalidInputBeforeServiceTC403(t *testing.T) {
	for _, args := range [][]string{{"Q001"}, {"E39-F04", "--limit", "101"}, {"E39-F04", "--offset", "-1"}, {"E39-F04", "--responder", "alice"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			stub := &questionListServiceStub{}
			withQuestionSvcOverride(t, stub)
			cmd := newQuestionBlockingForCommand()
			cmd.SetArgs(args)
			if err := cmd.Execute(); err == nil {
				t.Fatal("TC-403 command error = nil, want validation rejection")
			}
			if stub.blockingForIn != nil {
				t.Fatal("TC-403 focused service called for invalid input")
			}
		})
	}
}

// TC-405: the distinct full command carries an explicit actor and cannot be
// confused with the compact get path.
func TestQuestionFullCommandTC405(t *testing.T) {
	stub := &questionListServiceStub{}
	withQuestionSvcOverride(t, stub)
	cmd := newQuestionFullCommand()
	cmd.SetArgs([]string{"Q001", "--actor", "alice"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("TC-405 command error = %v", err)
	}
	if got, want := *stub.fullReadIn, (fullReadInput{"Q001", "alice"}); got != want {
		t.Fatalf("TC-405 service input = %#v, want %#v", got, want)
	}
}

func TestQuestionFullRejectsBlankActorBeforeServiceTC405(t *testing.T) {
	stub := &questionListServiceStub{}
	withQuestionSvcOverride(t, stub)
	cmd := newQuestionFullCommand()
	cmd.SetArgs([]string{"Q001", "--actor", "  "})
	if err := cmd.Execute(); err == nil {
		t.Fatal("TC-405 command error = nil, want missing actor rejection")
	}
	if stub.fullReadIn != nil {
		t.Fatal("TC-405 service called for blank actor")
	}
}

// TC-405: policy identities must reach the service unchanged. Whitespace is
// malformed input, not a second spelling of an authorized responder.
func TestQuestionFullRejectsUntrimmedActorBeforeServiceTC405(t *testing.T) {
	stub := &questionListServiceStub{}
	withQuestionSvcOverride(t, stub)
	cmd := newQuestionFullCommand()
	cmd.SetArgs([]string{"Q001", "--actor", " alice "})
	if err := cmd.Execute(); err == nil {
		t.Fatal("TC-405 command error = nil, want untrimmed actor rejection")
	}
	if stub.fullReadIn != nil {
		t.Fatal("TC-405 service called for untrimmed actor")
	}
}

// TC-405: every malformed policy identity must be rejected by the CLI before
// the service seam, matching the HTTP full-read boundary.
func TestQuestionFullRejectsMalformedActorBeforeServiceTC405(t *testing.T) {
	for _, actor := range []string{string([]byte{0xff}), strings.Repeat("a", 257), "bearer token"} {
		t.Run("malformed policy identity", func(t *testing.T) {
			stub := &questionListServiceStub{}
			withQuestionSvcOverride(t, stub)
			cmd := newQuestionFullCommand()
			cmd.SetArgs([]string{"Q001", "--actor", actor})
			if err := cmd.Execute(); err == nil {
				t.Fatal("TC-405 command error = nil, want malformed actor rejection")
			}
			if stub.fullReadIn != nil {
				t.Fatal("TC-405 service called for malformed actor")
			}
		})
	}
}

// The question-specific tag command is a generic registration: it must route
// through the Question service to resolve the durable entity ID before the
// shared tag service can attach a tag.
func TestQuestionRegistrationIncludesTagTransportAndResolvesCanonicalID(t *testing.T) {
	withQuestionSvcOverride(t, questionResolverStub{question: &models.Question{
		BaseEntity: models.BaseEntity{ID: 41, Key: "Q001"},
	}})

	if command, _, err := questionCmd.Find([]string{"tag", "add"}); err != nil || command == nil {
		t.Fatalf("question tag transport not registered: command=%v err=%v", command, err)
	}
	got, err := resolveQuestionID(context.Background(), "q001")
	if err != nil || got != 41 {
		t.Fatalf("resolveQuestionID(q001) = %d, %v; want 41, nil", got, err)
	}
}

// Question is a workflow entity, so the administrative preview command must
// use the same Question-specific placeholder path as keyed dispatch. Falling
// back to the generic key-only map leaves the responder prompt incomplete even
// though `shark next Q001` is correctly registered.
func TestQuestionWorkflowShowActionUsesQuestionCallerPath(t *testing.T) {
	contextData := `{"question_state":{"resolution_owner":"owner","responders":[{"identity":"alice","status":"pending"}]}}`
	withQuestionSvcOverride(t, questionResolverStub{question: &models.Question{
		Status:     models.QuestionStatusOpen,
		Summary:    "Need a decision",
		Requester:  "requester",
		BaseEntity: models.BaseEntity{Key: "Q001", Title: "Question", ContextData: &contextData},
	}})

	placeholders, err := lookupEntityPlaceholders(context.Background(), "Q001", "question")
	if err != nil {
		t.Fatalf("lookupEntityPlaceholders(question) error = %v", err)
	}
	for key, want := range map[string]string{
		"key":               "Q001",
		"summary":           "Need a decision",
		"requester":         "requester",
		"current_responder": "alice",
	} {
		if got := placeholders[key]; got != want {
			t.Errorf("Question show-action placeholder %q = %q, want %q", key, got, want)
		}
	}
}

// TC-108 executes the real registered Question Cobra command tree. A Find
// assertion only proves registration; Execute proves Cobra's production
// argument and flag parsing reaches the finite workflow transport and that its
// public result remains the metadata-only Question projection.
func TestQuestionWorkflowOperationsExecuteFiniteTransport_TC108(t *testing.T) {
	originalJSON := cli.GlobalConfig.JSON
	t.Cleanup(func() { cli.GlobalConfig.JSON = originalJSON })
	cli.GlobalConfig.JSON = true

	contextData := `{"question_state":{"secret":"must-not-cross-cli"}}`
	for _, tc := range []struct {
		name        string
		args        []string
		missingArgs [][]string
		assertInput func(*testing.T, *questionListServiceStub)
	}{
		{
			name: "configure-workflow",
			args: []string{"configure-workflow", "q001", "--resolution-owner", "owner", "--responder", "alice", "--responder", "bob"},
			missingArgs: [][]string{
				{"configure-workflow", "q001", "--responder", "alice"},
				{"configure-workflow", "q001", "--resolution-owner", "owner"},
			},
			assertInput: func(t *testing.T, stub *questionListServiceStub) {
				t.Helper()
				if got := stub.configureIn; got == nil || got.Key != "Q001" || got.ResolutionOwner != "owner" || strings.Join(got.Responders, ",") != "alice,bob" {
					t.Fatalf("ConfigureWorkflow input = %#v configured=%v", got, stub.configured)
				}
			},
		},
		{
			name: "respond",
			args: []string{"respond", "q001", "--session", "session-a", "--responder", "alice", "--summary", "approved", "--evidence-pointer", "docs/spec.md"},
			missingArgs: [][]string{
				{"respond", "q001", "--responder", "alice", "--summary", "approved", "--evidence-pointer", "docs/spec.md"},
				{"respond", "q001", "--session", "session-a", "--summary", "approved", "--evidence-pointer", "docs/spec.md"},
				{"respond", "q001", "--session", "session-a", "--responder", "alice", "--evidence-pointer", "docs/spec.md"},
				{"respond", "q001", "--session", "session-a", "--responder", "alice", "--summary", "approved"},
			},
			assertInput: func(t *testing.T, stub *questionListServiceStub) {
				t.Helper()
				if got := stub.responseIn; got == nil || got.Key != "Q001" || got.SessionID != "session-a" || got.Responder != "alice" || got.Summary != "approved" || got.EvidencePointer != "docs/spec.md" {
					t.Fatalf("RecordResponse input = %#v", got)
				}
			},
		},
		{
			name: "resolve",
			args: []string{"resolve", "q001", "--owner", "owner", "--resolution-kind", "feature_change", "--resolution-pointer", "E39-F03"},
			missingArgs: [][]string{
				{"resolve", "q001", "--resolution-kind", "feature_change", "--resolution-pointer", "E39-F03"},
				{"resolve", "q001", "--owner", "owner", "--resolution-pointer", "E39-F03"},
				{"resolve", "q001", "--owner", "owner", "--resolution-kind", "feature_change"},
			},
			assertInput: func(t *testing.T, stub *questionListServiceStub) {
				t.Helper()
				if got := stub.resolveIn; got == nil || got.Key != "Q001" || got.Owner != "owner" || got.Kind != "feature_change" || got.Pointer != "E39-F03" {
					t.Fatalf("Resolve input = %#v", got)
				}
			},
		},
		{
			name: "withdraw",
			args: []string{"withdraw", "q001", "--owner", "owner", "--reason", "obsolete"},
			missingArgs: [][]string{
				{"withdraw", "q001", "--reason", "obsolete"},
				{"withdraw", "q001", "--owner", "owner"},
			},
			assertInput: func(t *testing.T, stub *questionListServiceStub) {
				t.Helper()
				if got := stub.withdrawIn; got == nil || got.Key != "Q001" || got.Owner != "owner" || got.Reason != "obsolete" {
					t.Fatalf("Withdraw input = %#v", got)
				}
			},
		},
		{
			name: "supersede",
			args: []string{"supersede", "q001", "--owner", "owner", "--reason", "obsolete", "--superseded-by", "q002"},
			missingArgs: [][]string{
				{"supersede", "q001", "--reason", "obsolete", "--superseded-by", "q002"},
				{"supersede", "q001", "--owner", "owner", "--superseded-by", "q002"},
				{"supersede", "q001", "--owner", "owner", "--reason", "obsolete"},
			},
			assertInput: func(t *testing.T, stub *questionListServiceStub) {
				t.Helper()
				if got := stub.supersedeIn; got == nil || got.Key != "Q001" || got.Owner != "owner" || got.Reason != "obsolete" || got.SupersededBy != "Q002" {
					t.Fatalf("Supersede input = %#v", got)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &questionListServiceStub{}
			withQuestionSvcOverride(t, stub)
			stubQuestion := &models.Question{BaseEntity: models.BaseEntity{Key: "Q001", Title: "Question", ContextData: &contextData}}
			stub.question = stubQuestion

			output, err := executeQuestionCommand(t, tc.args...)
			if err != nil {
				t.Fatalf("question %s Execute() error = %v", tc.name, err)
			}
			tc.assertInput(t, stub)
			assertQuestionMetadataProjection(t, output)

			for _, missingArgs := range tc.missingArgs {
				stub = &questionListServiceStub{}
				withQuestionSvcOverride(t, stub)
				if _, err := executeQuestionCommand(t, missingArgs...); err == nil || !strings.Contains(err.Error(), "is required") {
					t.Fatalf("question %s missing required flag Execute() error = %v", tc.name, err)
				}
				assertQuestionWorkflowNotCalled(t, stub)
			}

			stub = &questionListServiceStub{}
			withQuestionSvcOverride(t, stub)
			unsupportedArgs := append(append([]string{}, tc.args...), "--not-a-question-flag=value")
			if _, err := executeQuestionCommand(t, unsupportedArgs...); err == nil || !strings.Contains(err.Error(), "unknown flag") {
				t.Fatalf("question %s unsupported input Execute() error = %v", tc.name, err)
			}
			assertQuestionWorkflowNotCalled(t, stub)
		})
	}
}

// executeQuestionCommand intentionally drives the registered production
// command tree, rather than a test-local command or a direct RunE invocation.
func executeQuestionCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := cli.RootCmd
	previousPreRun := root.PersistentPreRunE
	previousPostRun := root.PersistentPostRunE
	previousSilenceErrors := root.SilenceErrors
	previousSilenceUsage := root.SilenceUsage
	root.PersistentPreRunE = nil
	root.PersistentPostRunE = nil
	root.SilenceErrors = true
	root.SilenceUsage = true
	// Drive the same root-level JSON flag a caller supplies in production. Cobra
	// resets the shared bound value while parsing each execution, so setting the
	// package global once at the test boundary is not enough when this command
	// tree runs after other CLI tests.
	root.SetArgs(append([]string{"--json", "question"}, args...))
	defer func() {
		root.PersistentPreRunE = previousPreRun
		root.PersistentPostRunE = previousPostRun
		root.SilenceErrors = previousSilenceErrors
		root.SilenceUsage = previousSilenceUsage
		root.SetArgs(nil)
	}()
	defer resetQuestionCommandFlags(t, questionCmd)
	return captureStdoutForTest(t, root.Execute)
}

func resetQuestionCommandFlags(t *testing.T, command *cobra.Command) {
	t.Helper()
	command.Flags().VisitAll(func(flag *pflag.Flag) {
		if !flag.Changed {
			return
		}
		if slice, ok := flag.Value.(pflag.SliceValue); ok {
			if err := slice.Replace(nil); err != nil {
				t.Fatalf("reset --%s: %v", flag.Name, err)
			}
		} else if err := flag.Value.Set(flag.DefValue); err != nil {
			t.Fatalf("reset --%s: %v", flag.Name, err)
		}
		flag.Changed = false
	})
	for _, child := range command.Commands() {
		resetQuestionCommandFlags(t, child)
	}
}

func assertQuestionMetadataProjection(t *testing.T, output string) {
	t.Helper()
	var projection models.QuestionProjection
	if err := json.Unmarshal([]byte(output), &projection); err != nil {
		t.Fatalf("metadata JSON = %q: %v", output, err)
	}
	if projection.Key != "Q001" || projection.Title != "Question" {
		t.Fatalf("metadata projection = %#v", projection)
	}
	if strings.Contains(output, "question_state") || strings.Contains(output, "must-not-cross-cli") || strings.Contains(output, "context_data") {
		t.Fatalf("metadata output leaked persisted workflow state: %s", output)
	}
}

func assertQuestionWorkflowNotCalled(t *testing.T, stub *questionListServiceStub) {
	t.Helper()
	if stub.configured || stub.responded || stub.resolved || stub.withdrawn || stub.superseded {
		t.Fatal("Question workflow service was called for rejected CLI input")
	}
}

type questionResolverStub struct{ question *models.Question }

func (s questionResolverStub) CreateQuestion(context.Context, services.CreateQuestionInput) (*models.Question, error) {
	return s.question, nil
}
func (s questionResolverStub) GetQuestion(context.Context, string) (*models.Question, error) {
	return s.question, nil
}
func (s questionResolverStub) ListQuestions(context.Context, services.QuestionListFilter) ([]*models.Question, error) {
	return nil, nil
}
func (s questionResolverStub) UpdateQuestion(context.Context, string, services.QuestionUpdates) (*models.Question, error) {
	return s.question, nil
}
func (s questionResolverStub) DeleteQuestion(context.Context, string) error { return nil }
func (s questionResolverStub) GetNextStatus(_ context.Context, key string) (*services.NextStatusInfo, error) {
	return &services.NextStatusInfo{EntityType: models.EntityTypeQuestion, EntityKey: key}, nil
}
func (s questionResolverStub) TransitionStatus(_ context.Context, key, target string, _ services.TransitionOptions) (*services.TransitionResult, error) {
	return &services.TransitionResult{EntityType: models.EntityTypeQuestion, EntityKey: key, ToStatus: target}, nil
}
func (s questionResolverStub) ConfigureWorkflow(context.Context, services.ConfigureWorkflowInput) (*models.Question, error) {
	return s.question, nil
}
func (s questionResolverStub) RecordResponse(context.Context, services.RecordQuestionResponseInput) (*models.Question, error) {
	return s.question, nil
}
func (s questionResolverStub) Resolve(context.Context, services.ResolveQuestionInput) (*models.Question, error) {
	return s.question, nil
}
func (s questionResolverStub) Withdraw(context.Context, services.WithdrawQuestionInput) (*models.Question, error) {
	return s.question, nil
}
func (s questionResolverStub) Supersede(context.Context, services.SupersedeQuestionInput) (*models.Question, error) {
	return s.question, nil
}
func (s questionResolverStub) ListOpenQuestionsByResponder(context.Context, string, int, int) ([]models.QuestionProjection, error) {
	return nil, nil
}
func (s questionResolverStub) ListQuestionsBlocking(context.Context, models.EntityType, string, int, int) ([]*services.QuestionBlock, error) {
	return nil, nil
}
func (s questionResolverStub) ReadQuestionFull(context.Context, string, string) (*models.QuestionFullProjection, error) {
	return nil, nil
}
