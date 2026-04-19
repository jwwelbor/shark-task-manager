package commands

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/jwwelbor/shark-task-manager/internal/cli"
	viewerserver "github.com/jwwelbor/shark-task-manager/internal/viewer/server"
	"github.com/spf13/cobra"
)

// webPort is the default starting port for shark web.
var webPort int

// webNoOpen skips launching a browser when true.
var webNoOpen bool

// webCmd represents the `shark web` command.
var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Launch the Shark web dashboard",
	Long: `Start a local web server and open the Shark task-viewer dashboard in
your default browser.

Port selection priority (highest to lowest):
  1. --port flag — use exactly this port; fail if busy
  2. web.port in .sharkconfig.json — use this port; fail if busy
  3. Default: bind to 127.0.0.1:7777 (falling back to 7778–7790 if busy)

Configure a persistent default port in .sharkconfig.json:
  { "web": { "port": 8080 } }`,
	Example: `  shark web                   # Start on first free port from 7777
  shark web --port 8080       # Use port 8080 exactly (fail if busy)
  shark web --no-open         # Start server but do not open browser`,
	RunE: runWeb,
}

func init() {
	webCmd.Flags().IntVar(&webPort, "port", 7777, "Port to listen on; when set explicitly, fails if the port is busy instead of falling back")
	webCmd.Flags().BoolVar(&webNoOpen, "no-open", false, "Do not open a browser after the server starts")
	cli.RootCmd.AddCommand(webCmd)
}

// findFreePort tries each port in [start, end] and returns the first one that
// is not already in use.  The test listener is closed before returning so the
// caller can immediately bind the chosen port.
func findFreePort(start, end int) (int, error) {
	for port := start; port <= end; port++ {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			// Port is busy — try the next one.
			continue
		}
		// Port is free; close the probe listener before returning.
		if closeErr := ln.Close(); closeErr != nil {
			return 0, fmt.Errorf("failed to close probe listener on port %d: %w", port, closeErr)
		}
		return port, nil
	}
	return 0, fmt.Errorf("no free port found in range %d–%d", start, end)
}

// openBrowserForOS launches the system default browser for the given URL on
// the specified OS.  Separating the OS selection from runtime.GOOS makes the
// function unit-testable.
// The command is started asynchronously (fire-and-forget) so the caller is
// not blocked waiting for the browser to exit.
// Returns an error if the OS is unsupported or the command fails to start;
// this error is non-fatal at the call site.
func openBrowserForOS(goos, url string) error {
	var cmd *exec.Cmd
	switch goos {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported OS: %s", goos)
	}
	return cmd.Start()
}

// openBrowser launches the system default browser using the current OS.
func openBrowser(url string) error {
	return openBrowserForOS(runtime.GOOS, url)
}

func loadConfigPort() int {
	cfg, err := cli.GetConfig()
	if err != nil {
		return 0
	}
	return cfg.GetWebPort()
}

func runWeb(cmd *cobra.Command, args []string) error {
	// --- 1. Find a free port -------------------------------------------------

	var port int
	var err error
	if cmd.Flags().Changed("port") {
		// User explicitly specified a port — fail fast if busy rather than
		// silently falling back to another port.
		port, err = findFreePort(webPort, webPort)
		if err != nil {
			return fmt.Errorf("port %d is already in use", webPort)
		}
	} else if cfgPort := loadConfigPort(); cfgPort > 0 {
		port, err = findFreePort(cfgPort, cfgPort)
		if err != nil {
			return fmt.Errorf("port %d (from .sharkconfig.json web.port) is already in use", cfgPort)
		}
	} else {
		port, err = findFreePort(webPort, webPort+13)
		if err != nil {
			return fmt.Errorf("could not find a free port: %w", err)
		}
	}

	// --- 2. Obtain DB connection ---------------------------------------------

	db, err := cli.GetDB(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// --- 3. Resolve project root ---------------------------------------------

	// The viewer serves files from disk (Files tab) relative to the project
	// root. Without this, the server would fall back to the process CWD and
	// fail to locate docs/ when `shark web` is launched from a subdirectory.
	projectRoot, err := cli.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	// --- 4. Build server options and ready channel ---------------------------

	readyCh := make(chan struct{})

	opts := viewerserver.Options{
		Addr:        fmt.Sprintf("127.0.0.1:%d", port),
		DB:          db,
		ProjectRoot: projectRoot,
		Ready:       readyCh,
	}

	// --- 5. Start server in a goroutine, wait for ready ---------------------

	// Create a cancellable context so we can shut down on OS signal.
	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srvErr := make(chan error, 1)
	go func() {
		srvErr <- viewerserver.StartServer(ctx, opts)
	}()

	// Block until the server signals it is ready to accept connections, or
	// until it fails before becoming ready.
	select {
	case err := <-srvErr:
		// Server exited before becoming ready.
		if err != nil {
			return fmt.Errorf("server failed to start: %w", err)
		}
		return nil
	case <-readyCh:
		// Server is ready.
	}

	// --- 6. Print URL and hint -----------------------------------------------

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	fmt.Printf("Shark viewer running at %s\n", url)
	fmt.Println("Press Ctrl+C to stop")

	// --- 7. Optionally open browser ------------------------------------------

	if !webNoOpen {
		if err := openBrowser(url); err != nil {
			cli.Warning(fmt.Sprintf("Could not open browser: %v", err))
		}
	}

	// --- 8. Block until context is cancelled (Ctrl-C / SIGTERM) -------------

	select {
	case err := <-srvErr:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
	case <-ctx.Done():
		fmt.Println("\nShutting down...")
		// Wait for server goroutine to finish.
		if err := <-srvErr; err != nil {
			return fmt.Errorf("server error during shutdown: %w", err)
		}
	}

	return nil
}
