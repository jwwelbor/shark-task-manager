package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	"github.com/jwwelbor/shark-task-manager/internal/config"
	"github.com/jwwelbor/shark-task-manager/internal/services"
	"github.com/spf13/cobra"
)

// maintainerBootstrapServiceIface is the interface the set-password command
// depends on. It allows tests to inject a mock instead of the real
// *services.MaintainerBootstrapService.
//
// Spec reference: spec.md §2.5 F02-D4, §2.6 admin command tree.
type maintainerBootstrapServiceIface interface {
	SetPassword(ctx context.Context, plaintextPassword string) error
}

// adminMaintainerCmd is the "shark admin maintainer" parent command.
var adminMaintainerCmd = &cobra.Command{
	Use:   "maintainer",
	Short: "Manage maintainer password",
	Long: `Commands for managing the maintainer authorization gate.

The maintainer password protects destructive admin operations via a
sudo-style cache mechanism. Run 'shark admin maintainer set-password'
to configure the password stored in .sharkconfig.json.`,
}

// adminMaintainerSetPasswordCmd is the "shark admin maintainer set-password" command.
// It is constructed with the real bootstrap service; tests use newAdminMaintainerSetPasswordCmd
// to inject a mock.
var adminMaintainerSetPasswordCmd = newAdminMaintainerSetPasswordCmd(nil) // placeholder; real svc wired in init()

// newAdminMaintainerSetPasswordCmd constructs the cobra command with the given
// bootstrap service injected. Passing nil uses the real service (resolved lazily
// at runtime). Tests pass a mock to avoid I/O.
//
// Spec reference: spec.md §2.4 (CLI command surface), §2.6 (admin command tree).
func newAdminMaintainerSetPasswordCmd(svc maintainerBootstrapServiceIface) *cobra.Command {
	var (
		flagPassword      string
		flagPasswordStdin bool
	)

	cmd := &cobra.Command{
		Use:   "set-password",
		Short: "Set the maintainer password",
		Long: `Set the maintainer password by writing its SHA-256 hash into .sharkconfig.json.

The plaintext password is NEVER stored. Only the SHA-256 hex digest is
written to the config file. All other config keys are preserved.

Provide the password via --password, --password-stdin, or (if neither
flag is set and stdin is a terminal) an interactive prompt.

Examples:
  shark admin maintainer set-password --password "hunter2"
  echo "hunter2" | shark admin maintainer set-password --password-stdin`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve the password from the available sources.
			pwd, err := resolvePassword(flagPassword, flagPasswordStdin)
			if err != nil {
				return fmt.Errorf("set-password: %w", err)
			}

			if pwd == "" {
				return fmt.Errorf("set-password: password cannot be empty")
			}

			// Resolve the bootstrap service (real or injected mock).
			bootstrapSvc := svc
			if bootstrapSvc == nil {
				var svcErr error
				bootstrapSvc, svcErr = newRealBootstrapService()
				if svcErr != nil {
					return fmt.Errorf("set-password: %w", svcErr)
				}
			}

			// Delegate all orchestration to the service (thin-wrapper pattern).
			if err := bootstrapSvc.SetPassword(cmd.Context(), pwd); err != nil {
				return fmt.Errorf("set-password: %w", err)
			}

			// Report success without echoing the password or its hash (AC-12).
			cli.Success("Maintainer password configured successfully.")
			cli.Info("Run 'shark admin maintainer set-password' again to rotate the password.")
			return nil
		},
	}

	cmd.Flags().StringVar(&flagPassword, "password", "", "Plaintext password (not stored; SHA-256 hash is written to config)")
	cmd.Flags().BoolVar(&flagPasswordStdin, "password-stdin", false, "Read password from stdin (newline-terminated)")

	return cmd
}

func init() {
	adminMaintainerCmd.AddCommand(adminMaintainerSetPasswordCmd)
	adminCmd.AddCommand(adminMaintainerCmd)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// resolvePassword returns the password from the highest-priority source:
//  1. --password flag (if non-empty)
//  2. --password-stdin (read one line from os.Stdin)
//  3. Interactive prompt (not yet implemented; returns empty string)
//
// Spec reference: spec.md §2.4 (CLI command surface), AC-T1.
func resolvePassword(flagPassword string, fromStdin bool) (string, error) {
	if flagPassword != "" {
		return flagPassword, nil
	}

	if fromStdin {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			return strings.TrimRight(scanner.Text(), "\r\n"), nil
		}
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("reading password from stdin: %w", err)
		}
		return "", fmt.Errorf("no password provided via stdin")
	}

	// Interactive prompt is not yet implemented for this command.
	// Return empty string; the caller will reject it.
	return "", nil
}

// newRealBootstrapService constructs the real *services.MaintainerBootstrapService
// wired to the current project's .sharkconfig.json. Used by the production command
// path (when no mock is injected via newAdminMaintainerSetPasswordCmd).
//
// Spec reference: spec.md §2.5 F02-D4.
func newRealBootstrapService() (*services.MaintainerBootstrapService, error) {
	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}
	configPath := filepath.Join(projectRoot, ".sharkconfig.json")

	reader := &jsonConfigReader{path: configPath}
	writer := &jsonConfigWriter{path: configPath}
	return services.NewMaintainerBootstrapService(reader, writer), nil
}

// ---------------------------------------------------------------------------
// Config I/O adapters (implement services.ConfigReader / services.ConfigWriter)
// ---------------------------------------------------------------------------

// jsonConfigReader reads .sharkconfig.json as a raw map, preserving all fields.
type jsonConfigReader struct {
	path string
}

func (r *jsonConfigReader) Read() (map[string]interface{}, error) {
	mgr := config.NewManager(r.path)
	cfg, err := mgr.Load()
	if err != nil {
		return nil, err
	}
	if cfg == nil || cfg.RawData == nil {
		return nil, nil
	}
	return cfg.RawData, nil
}

// jsonConfigWriter writes a raw map back to .sharkconfig.json atomically.
type jsonConfigWriter struct {
	path string
}

func (w *jsonConfigWriter) Write(data map[string]interface{}) error {
	return writeConfig(w.path, data)
}
