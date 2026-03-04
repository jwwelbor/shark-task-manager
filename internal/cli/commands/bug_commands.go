package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/models"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// runBugGet retrieves and displays a specific bug.
// Called from runGet when a B### key is detected.
func runBugGet(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	bugKey := args[0]
	svc := cli.GetBugService()
	bug, err := svc.GetBug(ctx, bugKey)
	if err != nil {
		return fmt.Errorf("bug %s not found: %w", bugKey, err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(bug)
	}

	renderBugDetails(bug)
	return nil
}

// runBugCreate creates a new bug.
// Called from runCreate when "bug" entity type is given.
func runBugCreate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	if len(args) < 1 {
		return fmt.Errorf("bug title is required")
	}
	title := args[0]

	severityStr, _ := cmd.Flags().GetString("severity")
	if severityStr == "" {
		severityStr = "medium"
	}
	description, _ := cmd.Flags().GetString("description")
	linkedType, _ := cmd.Flags().GetString("linked-type")
	linkedKey, _ := cmd.Flags().GetString("linked-key")

	input := services.CreateBugInput{
		Title:            title,
		Severity:         models.BugSeverity(severityStr),
		Description:      description,
		LinkedEntityType: linkedType,
		LinkedEntityKey:  linkedKey,
	}

	svc := cli.GetBugService()
	bug, err := svc.CreateBug(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create bug: %w", err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(bug)
	}

	cli.Success(fmt.Sprintf("Created bug %s: %s", bug.Key, bug.Title))
	return nil
}

// runBugDelete deletes a bug by key.
// Called from runDelete when a B### key is detected.
func runBugDelete(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	bugKey := args[0]
	svc := cli.GetBugService()
	if err := svc.DeleteBug(ctx, bugKey); err != nil {
		return fmt.Errorf("failed to delete bug %s: %w", bugKey, err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]string{"deleted": bugKey})
	}

	cli.Success(fmt.Sprintf("Deleted bug %s", bugKey))
	return nil
}

// renderBugDetails prints a human-readable bug summary to stdout.
func renderBugDetails(bug *models.Bug) {
	cli.Info(fmt.Sprintf("Bug: %s", bug.Key))
	fmt.Printf("  Title:    %s\n", bug.Title)
	fmt.Printf("  Status:   %s\n", bug.Status)
	fmt.Printf("  Severity: %s\n", bug.Severity)
	if bug.Description != nil && *bug.Description != "" {
		fmt.Printf("  Desc:     %s\n", *bug.Description)
	}
	if bug.LinkedEntityType != nil && bug.LinkedEntityKey != nil {
		fmt.Printf("  Linked:   %s %s\n", *bug.LinkedEntityType, *bug.LinkedEntityKey)
	}
}
