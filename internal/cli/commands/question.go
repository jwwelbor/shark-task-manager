package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type questionServicer interface {
	CreateQuestion(context.Context, services.CreateQuestionInput) (*models.Question, error)
	GetQuestion(context.Context, string) (*models.Question, error)
	ListQuestions(context.Context, services.QuestionListFilter) ([]*models.Question, error)
	UpdateQuestion(context.Context, string, services.QuestionUpdates) (*models.Question, error)
	DeleteQuestion(context.Context, string) error
	TransitionStatus(context.Context, string, string, services.TransitionOptions) (*services.TransitionResult, error)
	GetNextStatus(context.Context, string) (*services.NextStatusInfo, error)
	ConfigureWorkflow(context.Context, services.ConfigureWorkflowInput) (*models.Question, error)
	RecordResponse(context.Context, services.RecordQuestionResponseInput) (*models.Question, error)
	Resolve(context.Context, services.ResolveQuestionInput) (*models.Question, error)
	Withdraw(context.Context, services.WithdrawQuestionInput) (*models.Question, error)
	Supersede(context.Context, services.SupersedeQuestionInput) (*models.Question, error)
	ListOpenQuestionsByResponder(context.Context, string, int, int) ([]models.QuestionProjection, error)
	ListQuestionsBlocking(context.Context, models.EntityType, string, int, int) ([]*services.QuestionBlock, error)
	ReadQuestionFull(context.Context, string, string) (*models.QuestionFullProjection, error)
}

var questionSvcOverride questionServicer

// questionListFlagNames is the full filtering and pagination contract for a
// Question list. In particular, the unified `shark list question` command
// shares flags with other entity types, so it must reject flags it cannot
// honor instead of returning an apparently-filtered unfiltered page.
var questionListFlagNames = map[string]struct{}{
	"status": {}, "requester": {}, "blocking": {}, "limit": {}, "offset": {},
}

func getQuestionService() questionServicer {
	if questionSvcOverride != nil {
		return questionSvcOverride
	}
	return cli.GetQuestionService()
}

// resolveQuestionID is the generic tag-command resolver for the concrete
// Question record. Keeping it on the Question service preserves the same
// thin transport boundary used by the other entity tag commands.
func resolveQuestionID(ctx context.Context, key string) (int64, error) {
	question, err := getQuestionService().GetQuestion(ctx, strings.ToUpper(key))
	if err != nil {
		return 0, err
	}
	return question.ID, nil
}

var questionCmd = &cobra.Command{Use: "question", Short: "Manage Questions", GroupID: "advanced", RunE: runQuestionRoot}
var questionCreateCmd = &cobra.Command{Use: "create <title>", Args: cobra.ExactArgs(1), RunE: runQuestionCreate}
var questionGetCmd = &cobra.Command{Use: "get <key>", Args: cobra.ExactArgs(1), RunE: runQuestionGet}
var questionListCmd = &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: runQuestionList}
var questionUpdateCmd = &cobra.Command{Use: "update <key>", Args: cobra.ExactArgs(1), RunE: runQuestionUpdate}
var questionDeleteCmd = &cobra.Command{Use: "delete <key>", Args: cobra.ExactArgs(1), RunE: runQuestionDelete}
var questionStatusCmd = &cobra.Command{Use: "status <key>", Args: cobra.ExactArgs(1), RunE: runQuestionStatus}
var questionConfigureWorkflowCmd = &cobra.Command{Use: "configure-workflow <key>", Args: cobra.ExactArgs(1), RunE: runQuestionConfigureWorkflow}
var questionRespondCmd = &cobra.Command{Use: "respond <key>", Args: cobra.ExactArgs(1), RunE: runQuestionRespond}
var questionResolveCmd = &cobra.Command{Use: "resolve <key>", Args: cobra.ExactArgs(1), RunE: runQuestionResolve}
var questionWithdrawCmd = &cobra.Command{Use: "withdraw <key>", Args: cobra.ExactArgs(1), RunE: runQuestionWithdraw}
var questionSupersedeCmd = &cobra.Command{Use: "supersede <key>", Args: cobra.ExactArgs(1), RunE: runQuestionSupersede}
var questionOpenByResponderCmd = newQuestionOpenByResponderCommand()
var questionBlockingForCmd = newQuestionBlockingForCommand()
var questionFullCmd = newQuestionFullCommand()

func runQuestionRoot(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}
	return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
}

func runQuestionCreate(cmd *cobra.Command, args []string) error {
	summary, _ := cmd.Flags().GetString("summary")
	requester, _ := cmd.Flags().GetString("requester")
	description, _ := cmd.Flags().GetString("description")
	blocking, _ := cmd.Flags().GetBool("blocking")
	status, _ := cmd.Flags().GetString("status")
	q, err := getQuestionService().CreateQuestion(cmd.Context(), services.CreateQuestionInput{Title: args[0], Summary: summary, Requester: requester, Description: description, Blocking: blocking, Status: status})
	if err != nil {
		return err
	}
	return outputQuestion(q, "Created Question")
}
func runQuestionGet(cmd *cobra.Command, args []string) error {
	q, err := getQuestionService().GetQuestion(cmd.Context(), strings.ToUpper(args[0]))
	if err != nil {
		return err
	}
	return outputQuestion(q, "")
}
func runQuestionStatus(cmd *cobra.Command, args []string) error { return runQuestionGet(cmd, args) }
func runQuestionList(cmd *cobra.Command, _ []string) error {
	if err := validateQuestionListFlags(cmd); err != nil {
		return err
	}
	filter := services.QuestionListFilter{Limit: 50}
	status, _ := cmd.Flags().GetString("status")
	requester, _ := cmd.Flags().GetString("requester")
	blocking, _ := cmd.Flags().GetString("blocking")
	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetInt("offset")
	if status != "" {
		s := models.QuestionStatus(status)
		filter.Status = &s
	}
	if requester != "" {
		filter.Requester = &requester
	}
	if blocking != "" {
		b, err := strconv.ParseBool(blocking)
		if err != nil {
			return fmt.Errorf("blocking must be boolean")
		}
		filter.Blocking = &b
	}
	filter.Limit = limit
	filter.Offset = offset
	qs, err := getQuestionService().ListQuestions(cmd.Context(), filter)
	if err != nil {
		return err
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(projectQuestions(qs))
	}
	for _, q := range qs {
		fmt.Printf("%s\t%s\t%s\n", q.Key, q.Status, q.Title)
	}
	return nil
}

func newQuestionOpenByResponderCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "open-by-responder <identity>", Args: cobra.ExactArgs(1), RunE: runQuestionOpenByResponder}
	cmd.Flags().Int("limit", 50, "Page limit (1-100)")
	cmd.Flags().Int("offset", 0, "Page offset")
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return cmd
}

func runQuestionOpenByResponder(cmd *cobra.Command, args []string) error {
	responder, err := requiredQuestionReadIdentity(args[0], "responder")
	if err != nil {
		return err
	}
	limit, offset, err := questionReadPage(cmd)
	if err != nil {
		return err
	}
	questions, err := getQuestionService().ListOpenQuestionsByResponder(cmd.Context(), responder, limit, offset)
	if err != nil {
		return err
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(questions)
	}
	for _, question := range questions {
		fmt.Printf("%s\t%s\t%s\n", question.Key, question.Status, question.Title)
	}
	return nil
}

func newQuestionBlockingForCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "blocking-for <entity-key>", Args: cobra.ExactArgs(1), RunE: runQuestionBlockingFor}
	cmd.Flags().Int("limit", 50, "Page limit (1-100)")
	cmd.Flags().Int("offset", 0, "Page offset")
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return cmd
}

func runQuestionBlockingFor(cmd *cobra.Command, args []string) error {
	rawTargetKey, err := requiredQuestionReadArgument(args[0], "blocking-for target")
	if err != nil {
		return err
	}
	targetKey := NormalizeKey(rawTargetKey)
	targetType, err := mapDetectedTypeToEntityType(DetectEntityType(targetKey))
	if err != nil {
		return fmt.Errorf("blocking-for target: %w", err)
	}
	if targetType == models.EntityTypeQuestion {
		return fmt.Errorf("blocking-for target must not be a Question")
	}
	limit, offset, err := questionReadPage(cmd)
	if err != nil {
		return err
	}
	blocks, err := getQuestionService().ListQuestionsBlocking(cmd.Context(), targetType, targetKey, limit, offset)
	if err != nil {
		return err
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(blocks)
	}
	for _, block := range blocks {
		fmt.Printf("%s\t%s\t%s\t%s\n", block.QuestionKey, block.Summary, block.ResolutionOwner, block.CurrentResponder)
	}
	return nil
}

func newQuestionFullCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "full <key>", Args: cobra.ExactArgs(1), RunE: runQuestionFull}
	cmd.Flags().String("actor", "", "Requesting responder or resolution owner")
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return cmd
}

func runQuestionFull(cmd *cobra.Command, args []string) error {
	actor, err := requiredQuestionReadIdentityFlag(cmd, "actor")
	if err != nil {
		return err
	}
	question, err := getQuestionService().ReadQuestionFull(cmd.Context(), strings.ToUpper(args[0]), actor)
	if err != nil {
		return err
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(question)
	}
	fmt.Printf("%s\t%s\t%s\n", question.Key, question.Status, question.Title)
	return nil
}

// requiredQuestionReadArgument preserves the supplied focused-read selector.
// Focused reads use identities and keys as policy/routing inputs, so accepting
// whitespace by normalizing it would select a different record than the caller
// supplied and diverge from the HTTP transport's finite validation boundary.
func requiredQuestionReadArgument(value, name string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value {
		return "", fmt.Errorf("%s is required and must be trimmed", name)
	}
	return value, nil
}

func requiredQuestionReadFlag(cmd *cobra.Command, name string) (string, error) {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return "", err
	}
	return requiredQuestionReadArgument(value, name)
}

func requiredQuestionReadIdentity(value, name string) (string, error) {
	identity, err := requiredQuestionReadArgument(value, name)
	if err != nil {
		return "", err
	}
	if err := services.ValidateQuestionReadIdentity(name, identity); err != nil {
		return "", err
	}
	return identity, nil
}

func requiredQuestionReadIdentityFlag(cmd *cobra.Command, name string) (string, error) {
	value, err := requiredQuestionReadFlag(cmd, name)
	if err != nil {
		return "", err
	}
	if err := services.ValidateQuestionReadIdentity(name, value); err != nil {
		return "", err
	}
	return value, nil
}

func questionReadPage(cmd *cobra.Command) (int, int, error) {
	limit, err := cmd.Flags().GetInt("limit")
	if err != nil {
		return 0, 0, err
	}
	offset, err := cmd.Flags().GetInt("offset")
	if err != nil {
		return 0, 0, err
	}
	if limit < 1 || limit > 100 {
		return 0, 0, fmt.Errorf("limit must be between 1 and 100")
	}
	if offset < 0 {
		return 0, 0, fmt.Errorf("offset must be non-negative")
	}
	return limit, offset, nil
}

func validateQuestionListFlags(cmd *cobra.Command) error {
	var unsupported string
	cmd.Flags().Visit(func(flag *pflag.Flag) {
		if unsupported != "" {
			return
		}
		// Root persistent flags control transport/configuration, not Question
		// filtering. Only a local list flag can violate this list contract.
		if cmd.LocalFlags().Lookup(flag.Name) == nil {
			return
		}
		if _, ok := questionListFlagNames[flag.Name]; !ok {
			unsupported = flag.Name
		}
	})
	if unsupported != "" {
		return fmt.Errorf("unsupported Question list flag --%s", unsupported)
	}
	return nil
}
func runQuestionUpdate(cmd *cobra.Command, args []string) error {
	u := services.QuestionUpdates{}
	if cmd.Flags().Changed("title") {
		v, _ := cmd.Flags().GetString("title")
		u.Title = &v
	}
	if cmd.Flags().Changed("summary") {
		v, _ := cmd.Flags().GetString("summary")
		u.Summary = &v
	}
	if cmd.Flags().Changed("requester") {
		v, _ := cmd.Flags().GetString("requester")
		u.Requester = &v
	}
	if cmd.Flags().Changed("description") {
		v, _ := cmd.Flags().GetString("description")
		u.Description = &v
	}
	if cmd.Flags().Changed("blocking") {
		v, _ := cmd.Flags().GetBool("blocking")
		u.Blocking = &v
	}
	q, err := getQuestionService().UpdateQuestion(cmd.Context(), strings.ToUpper(args[0]), u)
	if err != nil {
		return err
	}
	return outputQuestion(q, "Updated Question")
}
func runQuestionDelete(cmd *cobra.Command, args []string) error {
	key := strings.ToUpper(args[0])
	if err := getQuestionService().DeleteQuestion(cmd.Context(), key); err != nil {
		return err
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]string{"deleted": key})
	}
	cli.Success("Deleted Question " + key)
	return nil
}

func requiredQuestionString(cmd *cobra.Command, name string) (string, error) {
	value, _ := cmd.Flags().GetString(name)
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("--%s is required", name)
	}
	return value, nil
}

func runQuestionConfigureWorkflow(cmd *cobra.Command, args []string) error {
	owner, err := requiredQuestionString(cmd, "resolution-owner")
	if err != nil {
		return err
	}
	responders, _ := cmd.Flags().GetStringSlice("responder")
	if len(responders) == 0 {
		return fmt.Errorf("--responder is required")
	}
	q, err := getQuestionService().ConfigureWorkflow(cmd.Context(), services.ConfigureWorkflowInput{Key: strings.ToUpper(args[0]), ResolutionOwner: owner, Responders: responders})
	if err != nil {
		return err
	}
	return outputQuestion(q, "Configured Question workflow")
}

func runQuestionRespond(cmd *cobra.Command, args []string) error {
	session, err := requiredQuestionString(cmd, "session")
	if err != nil {
		return err
	}
	responder, err := requiredQuestionString(cmd, "responder")
	if err != nil {
		return err
	}
	summary, err := requiredQuestionString(cmd, "summary")
	if err != nil {
		return err
	}
	evidencePointer, err := requiredQuestionString(cmd, "evidence-pointer")
	if err != nil {
		return err
	}
	q, err := getQuestionService().RecordResponse(cmd.Context(), services.RecordQuestionResponseInput{Key: strings.ToUpper(args[0]), SessionID: session, Responder: responder, Summary: summary, EvidencePointer: evidencePointer})
	if err != nil {
		return err
	}
	return outputQuestion(q, "Recorded Question response")
}

func runQuestionResolve(cmd *cobra.Command, args []string) error {
	owner, err := requiredQuestionString(cmd, "owner")
	if err != nil {
		return err
	}
	kind, err := requiredQuestionString(cmd, "resolution-kind")
	if err != nil {
		return err
	}
	pointer, _ := cmd.Flags().GetString("resolution-pointer")
	if kind != "no_lasting_consequence" && strings.TrimSpace(pointer) == "" {
		return fmt.Errorf("--resolution-pointer is required")
	}
	q, err := getQuestionService().Resolve(cmd.Context(), services.ResolveQuestionInput{Key: strings.ToUpper(args[0]), Owner: owner, Kind: kind, Pointer: pointer})
	if err != nil {
		return err
	}
	return outputQuestion(q, "Resolved Question")
}

func runQuestionWithdraw(cmd *cobra.Command, args []string) error {
	owner, err := requiredQuestionString(cmd, "owner")
	if err != nil {
		return err
	}
	reason, err := requiredQuestionString(cmd, "reason")
	if err != nil {
		return err
	}
	q, err := getQuestionService().Withdraw(cmd.Context(), services.WithdrawQuestionInput{Key: strings.ToUpper(args[0]), Owner: owner, Reason: reason})
	if err != nil {
		return err
	}
	return outputQuestion(q, "Withdrew Question")
}

func runQuestionSupersede(cmd *cobra.Command, args []string) error {
	owner, err := requiredQuestionString(cmd, "owner")
	if err != nil {
		return err
	}
	reason, err := requiredQuestionString(cmd, "reason")
	if err != nil {
		return err
	}
	supersededBy, err := requiredQuestionString(cmd, "superseded-by")
	if err != nil {
		return err
	}
	q, err := getQuestionService().Supersede(cmd.Context(), services.SupersedeQuestionInput{Key: strings.ToUpper(args[0]), Owner: owner, Reason: reason, SupersededBy: strings.ToUpper(supersededBy)})
	if err != nil {
		return err
	}
	return outputQuestion(q, "Superseded Question")
}
func outputQuestion(q *models.Question, prefix string) error {
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(models.ProjectQuestion(q))
	}
	if prefix != "" {
		cli.Success(fmt.Sprintf("%s %s", prefix, q.Key))
	}
	fmt.Printf("%s\t%s\t%s\n", q.Key, q.Status, q.Title)
	return nil
}

func projectQuestions(questions []*models.Question) []models.QuestionProjection {
	projected := make([]models.QuestionProjection, 0, len(questions))
	for _, question := range questions {
		projected = append(projected, models.ProjectQuestion(question))
	}
	return projected
}

func init() {
	cli.RootCmd.AddCommand(questionCmd)
	questionCmd.AddCommand(questionCreateCmd, questionGetCmd, questionListCmd, questionUpdateCmd, questionDeleteCmd, questionStatusCmd, questionConfigureWorkflowCmd, questionRespondCmd, questionResolveCmd, questionWithdrawCmd, questionSupersedeCmd, questionOpenByResponderCmd, questionBlockingForCmd, questionFullCmd)
	questionCmd.AddCommand(makeEntityTagCmd(models.EntityTypeQuestion, resolveQuestionID, nil))
	questionCreateCmd.Flags().String("summary", "", "Question summary")
	questionCreateCmd.Flags().String("requester", "", "Question requester")
	questionCreateCmd.Flags().String("description", "", "Question description")
	questionCreateCmd.Flags().Bool("blocking", false, "Question blocks progress")
	questionCreateCmd.Flags().String("status", "", "Initial status (draft only)")
	questionListCmd.Flags().String("status", "", "Exact status")
	questionListCmd.Flags().String("requester", "", "Exact requester")
	questionListCmd.Flags().String("blocking", "", "Boolean blocking filter")
	questionListCmd.Flags().Int("limit", 50, "Page limit (1-100)")
	questionListCmd.Flags().Int("offset", 0, "Page offset")
	questionUpdateCmd.Flags().String("title", "", "New title")
	questionUpdateCmd.Flags().String("summary", "", "New summary")
	questionUpdateCmd.Flags().String("requester", "", "New requester")
	questionUpdateCmd.Flags().String("description", "", "New description")
	questionUpdateCmd.Flags().Bool("blocking", false, "Set blocking")
	questionConfigureWorkflowCmd.Flags().String("resolution-owner", "", "Configured resolution owner")
	questionConfigureWorkflowCmd.Flags().StringSlice("responder", nil, "Ordered responder identity (repeatable)")
	questionRespondCmd.Flags().String("session", "", "Active Question claim session")
	questionRespondCmd.Flags().String("responder", "", "Current responder identity")
	questionRespondCmd.Flags().String("summary", "", "Bounded response summary")
	questionRespondCmd.Flags().String("evidence-pointer", "", "Local evidence pointer")
	questionResolveCmd.Flags().String("owner", "", "Configured resolution owner")
	questionResolveCmd.Flags().String("resolution-kind", "", "Resolution classification")
	questionResolveCmd.Flags().String("resolution-pointer", "", "Validated resolution destination")
	questionWithdrawCmd.Flags().String("owner", "", "Configured resolution owner")
	questionWithdrawCmd.Flags().String("reason", "", "Bounded withdrawal reason")
	questionSupersedeCmd.Flags().String("owner", "", "Configured resolution owner")
	questionSupersedeCmd.Flags().String("reason", "", "Bounded supersession reason")
	questionSupersedeCmd.Flags().String("superseded-by", "", "Existing superseding Question key")
}
