// Package projectdocs verifies Git provenance of project documents.
package projectdocs

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type gitDoc struct {
	root, repo, relative, path string
}

func (d gitDoc) git(ctx context.Context, root string, args ...string) ([]byte, error) {
	return gitCommand(ctx, append([]string{"--literal-pathspecs", "-c", "core.fsmonitor=false", "-C", root}, args...)...).Output()
}

// VerifyCommitted checks that content is the selected checkout's tracked,
// non-ignored, clean, committed regular document, including index hints.
func VerifyCommitted(ctx context.Context, root, path string, content []byte) error {
	if !filepath.IsLocal(path) || !strings.HasPrefix(filepath.ToSlash(filepath.Clean(path)), "docs/plan/") {
		return fmt.Errorf("document must be a project document under docs/plan")
	}
	d := gitDoc{root: root, path: path}
	if err := d.locate(ctx); err != nil {
		return contextError(ctx, err)
	}
	if err := d.checkTracked(ctx); err != nil {
		return contextError(ctx, err)
	}
	if err := d.checkCommitted(ctx, content); err != nil {
		return contextError(ctx, err)
	}
	return nil
}

func contextError(ctx context.Context, fallback error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fallback
}

func (d *gitDoc) locate(ctx context.Context) error {
	out, err := d.git(ctx, d.root, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("find document repository: %w", err)
	}
	d.repo, err = filepath.EvalSymlinks(strings.TrimSpace(string(out)))
	if err != nil {
		return fmt.Errorf("resolve document repository: %w", err)
	}
	project, err := filepath.EvalSymlinks(d.root)
	if err != nil {
		return fmt.Errorf("resolve document project: %w", err)
	}
	d.relative, err = filepath.Rel(d.repo, project)
	if err != nil || !filepath.IsLocal(d.relative) {
		return fmt.Errorf("document project is outside repository")
	}
	return nil
}

func (d gitDoc) checkTracked(ctx context.Context) error {
	if _, err := d.git(ctx, d.root, "ls-files", "--error-unmatch", "--", d.path); err != nil {
		return fmt.Errorf("document is not tracked")
	}
	flags, err := d.git(ctx, d.root, "ls-files", "-v", "-z", "--", d.path)
	if err != nil {
		return fmt.Errorf("cannot inspect document index flags: %w", err)
	}
	if err := ValidateIndexFlags(flags); err != nil {
		return err
	}
	err = gitCommand(ctx, "-c", "core.fsmonitor=false", "-C", d.root, "check-ignore", "--no-index", "-q", "--", d.path).Run()
	if err == nil {
		return fmt.Errorf("document is ignored")
	}
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
		return fmt.Errorf("cannot check document ignore status: %w", err)
	}
	return nil
}

// ValidateIndexFlags rejects Git hints that conceal tracked changes.
func ValidateIndexFlags(data []byte) error {
	if len(data) > 0 && data[len(data)-1] != 0 {
		return fmt.Errorf("malformed Git index flags")
	}
	for _, record := range bytes.Split(data, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		if len(record) < 3 || record[1] != ' ' {
			return fmt.Errorf("malformed Git index flags")
		}
		flag := record[0]
		if flag == 'S' || (flag >= 'a' && flag <= 'z') {
			return fmt.Errorf("tracked path %q has assume-unchanged or skip-worktree set; clear the flag before review", string(record[2:]))
		}
	}
	return nil
}

func (d gitDoc) checkCommitted(ctx context.Context, content []byte) error {
	path := filepath.ToSlash(filepath.Join(d.relative, d.path))
	object := "HEAD:" + path
	kind, err := d.git(ctx, d.root, "cat-file", "-t", object)
	if err != nil || strings.TrimSpace(string(kind)) != "blob" {
		return fmt.Errorf("document is not in the current commit")
	}
	committed, err := d.git(ctx, d.root, "rev-parse", "--verify", "--end-of-options", object)
	if err != nil {
		return fmt.Errorf("cannot resolve committed document")
	}
	staged, err := d.git(ctx, d.repo, "rev-parse", "--verify", "--end-of-options", ":"+path)
	if err != nil || !bytes.Equal(bytes.TrimSpace(staged), bytes.TrimSpace(committed)) {
		return fmt.Errorf("document index differs from the current commit")
	}
	if err := d.checkMode(ctx, path); err != nil {
		return err
	}
	return d.checkContent(ctx, content, committed)
}

func (d gitDoc) checkMode(ctx context.Context, path string) error {
	index, err := d.git(ctx, d.root, "ls-files", "--stage", "-z", "--", d.path)
	if err != nil {
		return fmt.Errorf("cannot inspect document index mode")
	}
	tree, err := d.git(ctx, d.repo, "ls-tree", "--full-tree", "-z", "HEAD", "--", path)
	i, t := strings.Fields(string(index)), strings.Fields(string(tree))
	if err != nil || len(i) < 3 || len(t) < 3 || i[2] != "0" || i[0] != t[0] || (t[0] != "100644" && t[0] != "100755") {
		return fmt.Errorf("document mode differs from the current commit")
	}
	return nil
}

func (d gitDoc) checkContent(ctx context.Context, content, committed []byte) error {
	cmd := gitCommand(ctx, "-c", "core.fsmonitor=false", "-C", d.root, "hash-object", "--path", d.path, "--stdin")
	cmd.Stdin = bytes.NewReader(content)
	actual, err := cmd.Output()
	if err != nil || !bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(committed)) {
		return fmt.Errorf("document differs from the current commit")
	}
	status, err := d.git(ctx, d.root, "status", "--porcelain=v1", "-z", "--", d.path)
	if err != nil || len(status) != 0 {
		return fmt.Errorf("document differs from the current commit")
	}
	return nil
}
