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
func runChangeCardGet(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	cardKey := args[0]
	svc := cli.GetChangeCardService()
	card, err := svc.GetChangeCard(ctx, cardKey)
	if err != nil {
		return fmt.Errorf("change-card %s not found: %w", cardKey, err)
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(card)
	}

	renderChangeCardDetails(card)
	return nil
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

// renderChangeCardDetails prints a human-readable change-card summary to stdout.
func renderChangeCardDetails(card *models.ChangeCard) {
	cli.Info(fmt.Sprintf("Change Card: %s", card.Key))
	fmt.Printf("  Title:        %s\n", card.Title)
	fmt.Printf("  Status:       %s\n", card.Status)
	fmt.Printf("  Priority:     %d\n", card.Priority)
	if card.RequestedBy != nil && *card.RequestedBy != "" {
		fmt.Printf("  Requested By: %s\n", *card.RequestedBy)
	}
	if card.AssignedTo != nil && *card.AssignedTo != "" {
		fmt.Printf("  Assigned To:  %s\n", *card.AssignedTo)
	}
	if card.Description != nil && *card.Description != "" {
		fmt.Printf("  Description:  %s\n", *card.Description)
	}
	if card.Justification != nil && *card.Justification != "" {
		fmt.Printf("  Justification: %s\n", *card.Justification)
	}
}
