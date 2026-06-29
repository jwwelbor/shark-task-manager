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
	Use:   "list",
	Short: "List bundled skills",
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
	Use:   "list",
	Short: "List bundled agents",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBundleContentList(cmd, services.BundleContentKindAgent)
	},
}

func init() {
	skillGetCmd.Flags().Bool("raw", false, "Return exact stored content without include resolution or frontmatter stripping")
	agentGetCmd.Flags().Bool("raw", false, "Return exact stored content without include resolution or frontmatter stripping")

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

	_, err = fmt.Fprint(os.Stdout, result.Content)
	return err
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

	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(entries)
	}

	for _, entry := range entries {
		var line string
		if entry.Description != "" {
			line = fmt.Sprintf("%s — %s\n", entry.Name, entry.Description)
		} else {
			line = entry.Name + "\n"
		}
		if _, err := fmt.Fprint(os.Stdout, line); err != nil {
			return err
		}
	}
	return nil
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
