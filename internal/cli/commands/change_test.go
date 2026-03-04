package commands

import (
	"testing"

	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// newTestChangeUpdateCmd builds a cobra.Command with the flags used by
// buildChangeCardUpdates.
func newTestChangeUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "update"}
	cmd.Flags().StringVar(&changeTitle, "title", "", "Title")
	cmd.Flags().StringVar(&changeDescription, "description", "", "Description")
	cmd.Flags().IntVar(&changePriority, "priority", 0, "Priority")
	cmd.Flags().StringVar(&changeRequestedBy, "requested-by", "", "Requested by")
	cmd.Flags().StringVar(&changeAssignedTo, "assigned-to", "", "Assigned to")
	return cmd
}

// TestBuildCreateChangeCardInput_Title verifies that the title is passed through.
func TestBuildCreateChangeCardInput_Title(t *testing.T) {
	// Reset global flag variables before use
	changeLinkKey = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""

	input := buildCreateChangeCardInput("Add dark mode toggle")

	if input.Title != "Add dark mode toggle" {
		t.Errorf("expected title %q, got %q", "Add dark mode toggle", input.Title)
	}
}

// TestBuildCreateChangeCardInput_LinkEpic verifies that a bare epic key (e.g. E07)
// is placed in EpicKey and not FeatureKey.
func TestBuildCreateChangeCardInput_LinkEpic(t *testing.T) {
	changeLinkKey = "E07"
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""
	defer func() { changeLinkKey = "" }()

	input := buildCreateChangeCardInput("Some change")

	if input.EpicKey != "E07" {
		t.Errorf("expected EpicKey %q, got %q", "E07", input.EpicKey)
	}
	if input.FeatureKey != "" {
		t.Errorf("expected empty FeatureKey, got %q", input.FeatureKey)
	}
}

// TestBuildCreateChangeCardInput_LinkFeature verifies that a key containing "-F"
// (e.g. E07-F03) is placed in FeatureKey and not EpicKey.
func TestBuildCreateChangeCardInput_LinkFeature(t *testing.T) {
	changeLinkKey = "E07-F03"
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""
	defer func() { changeLinkKey = "" }()

	input := buildCreateChangeCardInput("Some change")

	if input.FeatureKey != "E07-F03" {
		t.Errorf("expected FeatureKey %q, got %q", "E07-F03", input.FeatureKey)
	}
	if input.EpicKey != "" {
		t.Errorf("expected empty EpicKey, got %q", input.EpicKey)
	}
}

// TestBuildCreateChangeCardInput_Priority verifies that a non-zero priority is set.
func TestBuildCreateChangeCardInput_Priority(t *testing.T) {
	changeLinkKey = ""
	changeDescription = ""
	changePriority = 8
	changeRequestedBy = ""
	defer func() { changePriority = 0 }()

	input := buildCreateChangeCardInput("Priority change")

	if input.Priority != 8 {
		t.Errorf("expected Priority 8, got %d", input.Priority)
	}
}

// TestBuildCreateChangeCardInput_ZeroPriorityOmitted verifies that a zero priority
// is not forwarded (it remains 0 in the input, which is fine as the service handles
// the default).
func TestBuildCreateChangeCardInput_ZeroPriorityOmitted(t *testing.T) {
	changeLinkKey = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""

	input := buildCreateChangeCardInput("Some change")

	if input.Priority != 0 {
		t.Errorf("expected Priority 0 when flag not set, got %d", input.Priority)
	}
}

// TestBuildCreateChangeCardInput_AllFields verifies all optional fields together.
func TestBuildCreateChangeCardInput_AllFields(t *testing.T) {
	changeLinkKey = "E10"
	changeDescription = "Detailed description"
	changePriority = 5
	changeRequestedBy = "alice"
	defer func() {
		changeLinkKey = ""
		changeDescription = ""
		changePriority = 0
		changeRequestedBy = ""
	}()

	input := buildCreateChangeCardInput("Full change")

	want := services.CreateChangeCardInput{
		Title:       "Full change",
		Description: "Detailed description",
		EpicKey:     "E10",
		Priority:    5,
		RequestedBy: "alice",
	}

	if input.Title != want.Title {
		t.Errorf("Title: expected %q, got %q", want.Title, input.Title)
	}
	if input.Description != want.Description {
		t.Errorf("Description: expected %q, got %q", want.Description, input.Description)
	}
	if input.EpicKey != want.EpicKey {
		t.Errorf("EpicKey: expected %q, got %q", want.EpicKey, input.EpicKey)
	}
	if input.Priority != want.Priority {
		t.Errorf("Priority: expected %d, got %d", want.Priority, input.Priority)
	}
	if input.RequestedBy != want.RequestedBy {
		t.Errorf("RequestedBy: expected %q, got %q", want.RequestedBy, input.RequestedBy)
	}
}

// TestBuildChangeCardUpdates_NoFlags verifies that when no flags are changed,
// all update fields remain nil.
func TestBuildChangeCardUpdates_NoFlags(t *testing.T) {
	changeTitle = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""
	changeAssignedTo = ""

	cmd := newTestChangeUpdateCmd()
	updates := buildChangeCardUpdates(cmd)

	if updates.Title != nil {
		t.Errorf("expected nil Title, got %v", updates.Title)
	}
	if updates.Description != nil {
		t.Errorf("expected nil Description, got %v", updates.Description)
	}
	if updates.Priority != nil {
		t.Errorf("expected nil Priority, got %v", updates.Priority)
	}
	if updates.RequestedBy != nil {
		t.Errorf("expected nil RequestedBy, got %v", updates.RequestedBy)
	}
	if updates.AssignedTo != nil {
		t.Errorf("expected nil AssignedTo, got %v", updates.AssignedTo)
	}
}

// TestBuildChangeCardUpdates_TitleChanged verifies that a changed title flag
// results in a non-nil Title pointer.
func TestBuildChangeCardUpdates_TitleChanged(t *testing.T) {
	changeTitle = "New title"
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""
	changeAssignedTo = ""
	defer func() { changeTitle = "" }()

	cmd := newTestChangeUpdateCmd()
	// Simulate the user passing --title on the CLI
	_ = cmd.Flags().Set("title", "New title")

	updates := buildChangeCardUpdates(cmd)

	if updates.Title == nil {
		t.Fatal("expected non-nil Title")
	}
	if *updates.Title != "New title" {
		t.Errorf("expected Title %q, got %q", "New title", *updates.Title)
	}
	// Other fields should remain nil
	if updates.Description != nil {
		t.Errorf("expected nil Description, got %v", updates.Description)
	}
}

// TestBuildChangeCardUpdates_MultipleFlags verifies several changed flags at once.
func TestBuildChangeCardUpdates_MultipleFlags(t *testing.T) {
	changeTitle = "Updated title"
	changeDescription = "Updated description"
	changePriority = 7
	changeRequestedBy = "bob"
	changeAssignedTo = ""
	defer func() {
		changeTitle = ""
		changeDescription = ""
		changePriority = 0
		changeRequestedBy = ""
	}()

	cmd := newTestChangeUpdateCmd()
	_ = cmd.Flags().Set("title", "Updated title")
	_ = cmd.Flags().Set("description", "Updated description")
	_ = cmd.Flags().Set("priority", "7")
	_ = cmd.Flags().Set("requested-by", "bob")

	updates := buildChangeCardUpdates(cmd)

	if updates.Title == nil || *updates.Title != "Updated title" {
		t.Errorf("Title: expected %q, got %v", "Updated title", updates.Title)
	}
	if updates.Description == nil || *updates.Description != "Updated description" {
		t.Errorf("Description: expected %q, got %v", "Updated description", updates.Description)
	}
	if updates.Priority == nil || *updates.Priority != 7 {
		t.Errorf("Priority: expected 7, got %v", updates.Priority)
	}
	if updates.RequestedBy == nil || *updates.RequestedBy != "bob" {
		t.Errorf("RequestedBy: expected %q, got %v", "bob", updates.RequestedBy)
	}
	if updates.AssignedTo != nil {
		t.Errorf("AssignedTo: expected nil, got %v", updates.AssignedTo)
	}
}

// TestBuildChangeCardUpdates_AssignedTo verifies that the assigned-to flag works.
func TestBuildChangeCardUpdates_AssignedTo(t *testing.T) {
	changeTitle = ""
	changeDescription = ""
	changePriority = 0
	changeRequestedBy = ""
	changeAssignedTo = "carol"
	defer func() { changeAssignedTo = "" }()

	cmd := newTestChangeUpdateCmd()
	_ = cmd.Flags().Set("assigned-to", "carol")

	updates := buildChangeCardUpdates(cmd)

	if updates.AssignedTo == nil || *updates.AssignedTo != "carol" {
		t.Errorf("AssignedTo: expected %q, got %v", "carol", updates.AssignedTo)
	}
	// Other fields should remain nil
	if updates.Title != nil {
		t.Errorf("expected nil Title, got %v", updates.Title)
	}
}

// TestPrintChangeCardList_Empty verifies that an empty slice prints a message
// and does not panic.
func TestPrintChangeCardList_Empty(t *testing.T) {
	// Should not panic and should return nil
	err := printChangeCardList(nil)
	if err != nil {
		t.Errorf("expected nil error for empty list, got %v", err)
	}
}
