package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// readPriorityIntFromFlag returns the integer value of the --priority flag,
// reading it as either an int (entity-first commands like `shark change update`)
// or a string (unified `shark update <KEY>` dispatch). Returns an error if the
// string form cannot be parsed as an integer in 1-10.
//
// The unified dispatch registers --priority as a string (since epic accepts
// "low/medium/high"); change-cards and ideas accept numeric priority. This
// helper bridges the two registrations so per-entity update functions work
// under either entry point without hardcoding the flag type. (B031)
func readPriorityIntFromFlag(cmd *cobra.Command) (int, error) {
	flag := cmd.Flags().Lookup("priority")
	if flag == nil {
		return 0, fmt.Errorf("priority flag is not registered")
	}
	if flag.Value.Type() == "int" {
		return cmd.Flags().GetInt("priority")
	}
	// string-typed (unified dispatch): parse as numeric 1-10
	s, _ := cmd.Flags().GetString("priority")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("--priority value cannot be empty")
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid --priority value %q: expected an integer 1-10", s)
	}
	return n, nil
}

// readStringSliceFromFlag returns a []string for the given flag name,
// reading from either StringSlice (entity-first registration) or
// comma-separated String (unified dispatch registration). Empty input
// returns an empty slice. Whitespace around comma-separated items is
// trimmed and empty items are dropped. (B031)
func readStringSliceFromFlag(cmd *cobra.Command, name string) ([]string, error) {
	flag := cmd.Flags().Lookup(name)
	if flag == nil {
		return nil, fmt.Errorf("--%s flag is not registered", name)
	}
	if flag.Value.Type() == "stringSlice" {
		return cmd.Flags().GetStringSlice(name)
	}
	// string-typed (unified dispatch): split on commas
	raw, _ := cmd.Flags().GetString(name)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out, nil
}

var updateCmd = &cobra.Command{
	Use:   "update <KEY> [flags]",
	Short: "Update an epic, feature, task, bug, change, tech-debt, or idea",
	Long: `Update an entity by key. The entity type is auto-detected from the key format.

Key format detection:
  E##                        Epic
  E##-F## or F##             Feature
  E##-F##-### or T-E##-F##-### Task
  B###                       Bug
  C###                       Change card
  TD-###                     Tech-debt
  I-YYYY-MM-DD-##            Idea

Use 'shark status set' to change entity status.

Common flags (all entities):
  --title          New title
  --description    New description
  --order          New execution order
  --parallel       Set --order without renumbering siblings (preserves duplicate-order parallel groups; task & feature only)
  --key            Rename entity key
  --file           New file path
  --force          Force file reassignment
  --size           New size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL (use 'clear' to remove)
  --tag            Apply tag additively (repeatable; bug/change/td/idea)

Priority & agent flags:
  --priority       New priority (1-10 for tasks; low/medium/high for epics; 1-10 for change/idea)
  --agent          New agent type (task only)
  --depends-on     New dependency keys, comma-separated (task & idea)

Epic-specific flags:
  --business-value New business value: low, medium, high

Bug-specific flags:
  --severity       New severity: critical, high, medium, low (bug & tech-debt)

Tech-debt-specific flags:
  --category       New category (code-quality, architecture, dependency, testing, performance, documentation)
  --effort-estimate New effort estimate (e.g., XS, S, M, L, XL)

Change-card-specific flags:
  --requested-by   Update requester
  --assigned-to    Assign to a person

Idea-specific flags:
  --notes          Update notes
  --related-docs   Update related document paths (comma-separated)

Examples:
  shark update E07 --title="New Epic Title"
  shark update E07 --business-value=high
  shark update E07-F01 --title="New Feature Title" --file=docs/custom/feature.md
  shark update E07-F01 --key=E07-F02
  shark update E07-F01-001 --title="New Task Title" --priority=8
  shark update E07-F01-001 --agent=backend
  shark update E07-F01-001 --size=L
  shark update E07-F01-002 --order=1 --parallel    # join existing order=1 group as parallel work
  shark update B030 --severity=medium
  shark update TD-001 --severity=high --category=performance
  shark update C001 --requested-by="Alice" --assigned-to="Bob"
  shark update TD-001 --size=clear`,
	GroupID: "manage",
	Args:    cobra.ExactArgs(1),
	RunE:    runUpdate,
}

func init() {
	// Common flags (all entities)
	updateCmd.Flags().String("title", "", "New title")
	updateCmd.Flags().StringP("description", "d", "", "New description")
	updateCmd.Flags().Int("order", -1, "New execution order (-1=no change)")
	updateCmd.Flags().Bool("parallel", false, "Set --order without renumbering siblings (preserves duplicate-order parallel groups; task & feature only)")

	// Key rename
	updateCmd.Flags().String("key", "", "New key (must be unique, no spaces)")

	// File path flags
	updateCmd.Flags().String("file", "", "New file path (e.g., docs/custom/feature.md)")
	updateCmd.Flags().String("filename", "", "Alias for --file")
	updateCmd.Flags().String("path", "", "Alias for --file")
	_ = updateCmd.Flags().MarkHidden("filename")
	_ = updateCmd.Flags().MarkHidden("path")
	updateCmd.Flags().Bool("force", false, "Force reassignment if file already claimed")

	// Priority (string: "1"-"10" for tasks, "low"/"medium"/"high" for epics)
	updateCmd.Flags().StringP("priority", "p", "", "New priority (1-10 for tasks; low/medium/high for epics)")

	// Task-specific flags
	updateCmd.Flags().StringP("agent", "a", "", "New agent type (task only)")
	updateCmd.Flags().String("depends-on", "", "New dependency keys, comma-separated (task only)")

	// Epic-specific flags
	updateCmd.Flags().String("business-value", "", "New business value: low, medium, high (epic only)")

	// Size flag (all entities) — string form per Decision D4: accepts numeric (5)
	// or t-shirt label (L) and the literal "clear" sentinel for removal.
	updateCmd.Flags().String("size", "",
		"New size: 1|2|3|5|8|13 or XS|S|M|L|XL|XXL (use 'clear' to remove)")

	// Tag flag (bug/change/td/idea) — additive on update. Use
	// `shark <entity> tag rm <key> <tag>` to detach.
	updateCmd.Flags().StringSlice("tag", nil,
		"Tag to apply additively (repeatable; bug/change/td/idea). Empty = no change; use 'shark <entity> tag rm' to detach.")

	// Bug- and tech-debt-specific flags
	updateCmd.Flags().String("severity", "",
		"New severity: critical, high, medium, low (bug & tech-debt)")

	// Tech-debt-specific flags
	updateCmd.Flags().String("category", "",
		"New category (tech-debt only: code-quality, architecture, dependency, testing, performance, documentation)")
	updateCmd.Flags().String("effort-estimate", "",
		"New effort estimate (tech-debt only: e.g., XS, S, M, L, XL)")

	// Change-card-specific flags
	updateCmd.Flags().String("requested-by", "",
		"Update requester (change-card only)")
	updateCmd.Flags().String("assigned-to", "",
		"Assign to a person (change-card only)")

	// Idea-specific flags
	updateCmd.Flags().String("notes", "",
		"Update notes (idea only)")
	updateCmd.Flags().StringSlice("related-docs", nil,
		"Update related document paths (idea only; comma-separated)")

	// Deprecated aliases
	updateCmd.Flags().Int("execution-order", -1, "New execution order (-1=no change)")
	_ = updateCmd.Flags().MarkDeprecated("execution-order", "use --order instead")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	key := args[0]
	entityType := DetectEntityType(key)

	switch entityType {
	case "epic":
		return runEpicUpdate(cmd, args)
	case "feature":
		return runFeatureUpdate(cmd, args)
	case "task":
		return runTaskUpdate(cmd, args)
	case "bug":
		return runBugUpdate(cmd, args)
	case "change", "change_card":
		return runChangeUpdate(cmd, args)
	case "tech_debt":
		return runTdUpdate(cmd, args)
	case "idea":
		return runIdeaUpdate(cmd, args)
	default:
		return fmt.Errorf("cannot determine entity type from key: %s\nExpected format: E## (epic), E##-F## (feature), E##-F##-### (task), B### (bug), C### (change card), TD-### (tech-debt), or I-YYYY-MM-DD-## (idea)", key)
	}
}
