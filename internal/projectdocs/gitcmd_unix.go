//go:build !windows

package projectdocs

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// gitCommand cancels the process group so clean-filter children cannot keep
// Git's output pipes open after the caller's deadline.
func gitCommand(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = 500 * time.Millisecond
	return cmd
}
