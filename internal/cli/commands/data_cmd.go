package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/research"
	"github.com/spf13/cobra"
)

var dataCmd = &cobra.Command{
	Use:     "data",
	Short:   "Read project shark-data resources",
	GroupID: "inspect",
}

var dataGetCmd = &cobra.Command{
	Use:   "get <resource-path>",
	Short: "Print a supported shark-data resource",
	Long: `Read a supported shark-data resource without installing or changing the
project bundle.

Supported resources:
  research/recipes.yaml  The configured or embedded v2 research recipe catalog`,
	Args: cobra.ExactArgs(1),
	RunE: runDataGet,
}

type dataResourceResponse struct {
	Path    string `json:"path"`
	Source  string `json:"source"`
	Content string `json:"content"`
}

func init() {
	dataCmd.AddCommand(dataGetCmd)
	cli.RootCmd.AddCommand(dataCmd)
}

func runDataGet(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	resourcePath := args[0]
	if resourcePath != research.CatalogResourcePath {
		return fmt.Errorf("unsupported data resource %q; supported resource: %s", resourcePath, research.CatalogResourcePath)
	}

	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("locate project root: %w", err)
	}
	configPath, err := cli.GetConfigPath()
	if err != nil {
		return fmt.Errorf("locate project configuration: %w", err)
	}
	content, source, err := research.ReadCatalogFromConfigContext(ctx, configPath, projectRoot)
	if err != nil {
		return err
	}
	result := dataResourceResponse{
		Path:    resourcePath,
		Source:  string(source),
		Content: string(content),
	}
	if cli.GlobalConfig.JSON {
		return cli.OutputJSON(result)
	}
	if _, err := fmt.Fprint(os.Stdout, result.Content); err != nil {
		return fmt.Errorf("write data resource: %w", err)
	}
	return nil
}
