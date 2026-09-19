package commands

import (
	"fmt"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

var skillCmd = &cobra.Command{
	Use:     "skill",
	Short:   "Retrieve bundled skill content",
	GroupID: "inspect",
}

var skillGetCmd = &cobra.Command{
	Use:   "get <name> [relative-path]",
	Short: "Print bundled skill content",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBundleContentGet(cmd, services.BundleContentKindSkill, args)
	},
}

var skillListCmd = &cobra.Command{
	Use:   "list [--all]",
	Short: "List bundled skills (use --all for descriptions; JSON includes sources)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBundleContentList(cmd, services.BundleContentKindSkill)
	},
}

var agentCmd = &cobra.Command{
	Use:     "agent",
	Short:   "Retrieve bundled agent content",
	GroupID: "inspect",
}

var agentGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Print bundled agent content",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBundleContentGet(cmd, services.BundleContentKindAgent, args)
	},
}

var agentListCmd = &cobra.Command{
	Use:   "list [--all]",
	Short: "List bundled agents (use --all for descriptions; JSON includes sources)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBundleContentList(cmd, services.BundleContentKindAgent)
	},
}

func init() {
	skillGetCmd.Flags().Bool("raw", false, "Return exact stored content without include resolution or frontmatter stripping")
	agentGetCmd.Flags().Bool("raw", false, "Return exact stored content without include resolution or frontmatter stripping")
	skillListCmd.Flags().Bool("all", false, "Show descriptions; JSON output also includes source metadata")
	agentListCmd.Flags().Bool("all", false, "Show descriptions; JSON output also includes source metadata")

	skillCmd.AddCommand(skillListCmd, skillGetCmd)
	agentCmd.AddCommand(agentListCmd, agentGetCmd)

	cli.RootCmd.AddCommand(skillCmd)
	cli.RootCmd.AddCommand(agentCmd)
}

func runBundleContentGet(cmd *cobra.Command, kind services.BundleContentKind, args []string) error {
	root, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("locate project root: %w", err)
	}

	svc, err := services.NewBundleContentService(root)
	if err != nil {
		return err
	}

	relPath := ""
	if len(args) > 1 {
		relPath = args[1]
	}

	raw, err := getBundleContentRawFlag(cmd)
	if err != nil {
		return err
	}

	result, err := svc.Get(cmd.Context(), kind, args[0], relPath, services.BundleContentGetOptions{Raw: raw})
	if err != nil {
		return err
	}

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}

	if _, err := fmt.Fprint(os.Stdout, result.Content); err != nil {
		return fmt.Errorf("write bundle content: %w", err)
	}
	return nil
}

func runBundleContentList(cmd *cobra.Command, kind services.BundleContentKind) error {
	root, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("locate project root: %w", err)
	}

	svc, err := services.NewBundleContentService(root)
	if err != nil {
		return err
	}

	entries, err := svc.List(cmd.Context(), kind)
	if err != nil {
		return err
	}

	showAll, err := getBundleContentAllFlag(cmd)
	if err != nil {
		return err
	}

	return outputBundleContentList(entries, showAll)
}

func outputBundleContentList(entries []services.BundleContentEntry, showAll bool) error {
	if cli.GlobalConfig.JSON {
		return outputBundleContentListJSON(entries, showAll)
	}
	return printBundleContentList(entries, showAll)
}

func outputBundleContentListJSON(entries []services.BundleContentEntry, showAll bool) error {
	if showAll {
		return cli.OutputJSON(bundleContentListDetails(entries))
	}
	return cli.OutputJSON(bundleContentListSummaries(entries))
}

func printBundleContentList(entries []services.BundleContentEntry, showAll bool) error {
	for index, entry := range entries {
		if _, err := fmt.Fprintln(os.Stdout, colorBundleContentName(entry.Name)); err != nil {
			return fmt.Errorf("write bundle content name: %w", err)
		}
		if !showAll {
			continue
		}
		if entry.Description != "" {
			if _, err := fmt.Fprintln(os.Stdout, entry.Description); err != nil {
				return fmt.Errorf("write bundle content description: %w", err)
			}
		}
		if index < len(entries)-1 {
			if _, err := fmt.Fprintln(os.Stdout); err != nil {
				return fmt.Errorf("write bundle content separator: %w", err)
			}
		}
	}
	if !showAll {
		if _, err := fmt.Fprintln(os.Stdout, "Use --all to show descriptions."); err != nil {
			return fmt.Errorf("write bundle content hint: %w", err)
		}
	}
	return nil
}

type bundleContentListSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type bundleContentListDetail struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

func bundleContentListSummaries(entries []services.BundleContentEntry) []bundleContentListSummary {
	result := make([]bundleContentListSummary, 0, len(entries))
	for _, entry := range entries {
		result = append(result, bundleContentListSummary{
			Name:        entry.Name,
			Description: entry.Description,
		})
	}
	return result
}

func bundleContentListDetails(entries []services.BundleContentEntry) []bundleContentListDetail {
	result := make([]bundleContentListDetail, 0, len(entries))
	for _, entry := range entries {
		result = append(result, bundleContentListDetail{
			Name:        entry.Name,
			Description: entry.Description,
			Source:      entry.Source,
		})
	}
	return result
}

func colorBundleContentName(name string) string {
	if cli.GlobalConfig.NoColor {
		return name
	}
	return cli.ColorCyan + name + cli.ColorReset
}

func getBundleContentRawFlag(cmd *cobra.Command) (bool, error) {
	if cmd == nil || cmd.Flags().Lookup("raw") == nil {
		return false, nil
	}
	raw, err := cmd.Flags().GetBool("raw")
	if err != nil {
		return false, fmt.Errorf("read raw flag: %w", err)
	}
	return raw, nil
}

func getBundleContentAllFlag(cmd *cobra.Command) (bool, error) {
	if cmd == nil || cmd.Flags().Lookup("all") == nil {
		return false, nil
	}
	showAll, err := cmd.Flags().GetBool("all")
	if err != nil {
		return false, fmt.Errorf("read all flag: %w", err)
	}
	return showAll, nil
}
