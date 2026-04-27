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

// runChangeCardGet retrieves and displays a specific change-card.
// Called from runGet when a CC-### key is detected.
// Delegates to runChangeGet to avoid duplicating the enrichment and display logic.
func runChangeCardGet(cmd *cobra.Command, args []string) error {
	return runChangeGet(cmd, args)
}

// runChangeCardCreate creates a new change-card.
// Called from runCreate when "change" or "change-card" entity type is given.
func runChangeCardCreate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	if len(args) < 1 {
		return fmt.Errorf("change-card title is required")
	}
	title := args[0]

	description, _ := cmd.Flags().GetString("description")
	justification, _ := cmd.Flags().GetString("justification")
	requestedBy, _ := cmd.Flags().GetString("requested-by")
	epicKey, _ := cmd.Flags().GetString("epic")
	featureKey, _ := cmd.Flags().GetString("feature")

	input := services.CreateChangeCardInput{
		Title:         title,
		Description:   description,
		Justification: justification,
		RequestedBy:   requestedBy,
		EpicKey:       epicKey,
		FeatureKey:    featureKey,
	}

	svc := cli.GetChangeCardService()
	card, err := svc.CreateChangeCard(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create change-card: %w", err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(card)
	}

	cli.Success(fmt.Sprintf("Created change-card %s: %s", card.Key, card.Title))
	if fp := card.GetFilePath(); fp != "" {
		cli.Info(fmt.Sprintf("File: %s", fp))
	}
	return nil
}

// runChangeCardDelete deletes a change-card by key.
// Called from runDelete when a CC-### key is detected.
func runChangeCardDelete(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	cardKey := args[0]
	svc := cli.GetChangeCardService()
	if err := svc.DeleteChangeCard(ctx, cardKey); err != nil {
		return fmt.Errorf("failed to delete change-card %s: %w", cardKey, err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(map[string]string{"deleted": cardKey})
	}

	cli.Success(fmt.Sprintf("Deleted change-card %s", cardKey))
	return nil
}

// buildChangeCardBasicInfo assembles the key-value info table for change-card display.
func buildChangeCardBasicInfo(card *models.ChangeCard) [][]string {
	var info [][]string

	info = append(info, []string{"Title", card.Title})
	info = append(info, []string{"Status", string(card.Status)})
	info = append(info, []string{"Priority", fmt.Sprintf("%d", card.Priority)})

	if card.RequestedBy != nil && *card.RequestedBy != "" {
		info = append(info, []string{"Requested By", *card.RequestedBy})
	}
	if card.AssignedTo != nil && *card.AssignedTo != "" {
		info = append(info, []string{"Assigned To", *card.AssignedTo})
	}
	if card.FilePath != nil && *card.FilePath != "" {
		info = append(info, []string{"File", *card.FilePath})
	}
	if card.Description != nil && *card.Description != "" {
		info = append(info, []string{"Description", *card.Description})
	}
	if card.Justification != nil && *card.Justification != "" {
		info = append(info, []string{"Justification", *card.Justification})
	}
	// E07-F42 REQ-F-006: human display uses "<label> (<num>)" or omits the row entirely.
	if card.Size != nil {
		info = append(info, []string{"Size", formatSize(card.Size)})
	}
	info = append(info, []string{"Created", card.CreatedAt.Format(time.RFC3339)})
	info = append(info, []string{"Updated", card.UpdatedAt.Format(time.RFC3339)})

	return info
}
