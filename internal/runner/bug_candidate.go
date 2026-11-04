package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type bugCandidateScope struct {
	ProjectRoot      string   `json:"project_root"`
	BaseCommit       string   `json:"base_commit"`
	CandidateCommit  string   `json:"candidate_commit"`
	CandidateTree    string   `json:"candidate_tree"`
	DiffSHA256       string   `json:"diff_sha256"`
	ChangedFiles     []string `json:"changed_files"`
	ChangedFileCount int      `json:"changed_file_count"`
	Source           *struct {
		SourceRoot            string   `json:"source_root"`
		SourceCommit          string   `json:"source_commit"`
		CandidateCommit       string   `json:"candidate_commit"`
		CandidateTree         string   `json:"candidate_tree"`
		SelectedPaths         []string `json:"selected_paths"`
		SelectedIndexSHA256   string   `json:"selected_index_sha256"`
		SourceInventorySHA256 string   `json:"source_inventory_sha256"`
	} `json:"source_snapshot"`
}

// verifyBugCandidate checks the reviewed tree against the current source, including
// selected uncommitted files when the review used an isolated snapshot.
func verifyBugCandidate(ctx context.Context, root string, data []byte) error {
	var scope bugCandidateScope
	if err := json.Unmarshal(data, &scope); err != nil {
		return fmt.Errorf("decode review scope: %w", err)
	}
	git := func(env []string, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
		cmd.Env = env
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("git %s: %w", args[0], err)
		}
		return out, nil
	}
	run := func(args ...string) ([]byte, error) { return git(nil, args...) }
	if err := validateBugCandidateManifest(scope, run); err != nil {
		return err
	}
	if scope.Source == nil {
		return validateCleanBugCandidate(root, scope, run)
	}
	return validateBugSnapshot(root, scope, git, run)
}

func validateBugCandidateManifest(scope bugCandidateScope, run func(...string) ([]byte, error)) error {
	trim := func(b []byte) string { return strings.TrimSpace(string(b)) }
	isHash := func(s string) bool {
		if len(s) != 40 && len(s) != 64 {
			return false
		}
		_, err := hex.DecodeString(s)
		return err == nil
	}
	if !isHash(scope.BaseCommit) || !isHash(scope.CandidateCommit) || !isHash(scope.CandidateTree) {
		return fmt.Errorf("invalid candidate identity")
	}
	if _, err := run("cat-file", "-e", scope.CandidateCommit+"^{commit}"); err != nil {
		return fmt.Errorf("candidate commit is invalid: %w", err)
	}
	tree, err := run("rev-parse", scope.CandidateCommit+"^{tree}")
	if err != nil || trim(tree) != scope.CandidateTree {
		return fmt.Errorf("candidate tree mismatch")
	}
	diff, err := run("diff", "--no-ext-diff", "--binary", scope.BaseCommit, scope.CandidateCommit)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(diff)
	if hex.EncodeToString(hash[:]) != scope.DiffSHA256 || len(diff) == 0 {
		return fmt.Errorf("candidate diff mismatch")
	}
	names, err := run("diff", "--name-only", "-z", scope.BaseCommit, scope.CandidateCommit)
	if err != nil {
		return err
	}
	var paths []string
	for _, name := range bytes.Split(names, []byte{0}) {
		if len(name) > 0 {
			paths = append(paths, string(name))
		}
	}
	if len(paths) != scope.ChangedFileCount || len(paths) != len(scope.ChangedFiles) {
		return fmt.Errorf("candidate manifest mismatch")
	}
	for i := range paths {
		if paths[i] != scope.ChangedFiles[i] {
			return fmt.Errorf("candidate manifest mismatch")
		}
	}
	return nil
}

func validateCleanBugCandidate(root string, scope bugCandidateScope, run func(...string) ([]byte, error)) error {
	trim := func(b []byte) string { return strings.TrimSpace(string(b)) }
	subject, err := run("log", "-1", "--format=%s", scope.CandidateCommit)
	if err != nil {
		return err
	}
	if trim(subject) == "Shark isolated review snapshot v1" {
		return fmt.Errorf("snapshot source metadata missing")
	}
	a, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	b, err := filepath.EvalSymlinks(scope.ProjectRoot)
	if err != nil || a != b {
		return fmt.Errorf("review project mismatch")
	}
	head, err := run("rev-parse", "HEAD")
	if err != nil || trim(head) != scope.CandidateCommit {
		return fmt.Errorf("candidate commit is stale")
	}
	status, err := run("status", "--porcelain", "--untracked-files=all")
	if err != nil || len(status) != 0 {
		return fmt.Errorf("candidate worktree changed")
	}
	return nil
}

func validateBugSnapshot(root string, scope bugCandidateScope, git func([]string, ...string) ([]byte, error), run func(...string) ([]byte, error)) error {
	source := scope.Source
	trim := func(b []byte) string { return strings.TrimSpace(string(b)) }
	a, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	b, err := filepath.EvalSymlinks(source.SourceRoot)
	if err != nil || a != b {
		return fmt.Errorf("snapshot source mismatch")
	}
	if source.CandidateCommit != scope.CandidateCommit || source.CandidateTree != scope.CandidateTree || len(source.SelectedPaths) == 0 {
		return fmt.Errorf("snapshot identity mismatch")
	}
	for _, path := range source.SelectedPaths {
		if !filepath.IsLocal(path) || path == "." {
			return fmt.Errorf("invalid selected path")
		}
	}
	head, err := run("rev-parse", "HEAD")
	if err != nil || trim(head) != source.SourceCommit {
		return fmt.Errorf("snapshot source commit changed")
	}
	inventory, err := bugSourceInventory(root, run)
	if err != nil {
		return err
	}
	if inventory != source.SourceInventorySHA256 {
		return fmt.Errorf("source change inventory changed")
	}
	args := append([]string{"--literal-pathspecs", "diff", "--no-ext-diff", "--cached", "--binary", source.SourceCommit, "--"}, source.SelectedPaths...)
	staged, err := run(args...)
	if err != nil {
		return err
	}
	stagedHash := sha256.Sum256(staged)
	if hex.EncodeToString(stagedHash[:]) != source.SelectedIndexSHA256 {
		return fmt.Errorf("source index changed")
	}
	temp, err := os.MkdirTemp("", "shark-review-index-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(temp) }()
	env := append(os.Environ(), "GIT_INDEX_FILE="+filepath.Join(temp, "index"))
	if _, err = git(env, "read-tree", source.SourceCommit); err != nil {
		return err
	}
	args = append([]string{"--literal-pathspecs", "add", "-A", "--"}, source.SelectedPaths...)
	if _, err = git(env, args...); err != nil {
		return err
	}
	actual, err := git(env, "write-tree")
	if err != nil || trim(actual) != scope.CandidateTree {
		return fmt.Errorf("source candidate changed")
	}
	return nil
}

func bugSourceInventory(root string, git func(...string) ([]byte, error)) (string, error) {
	h := sha256.New()
	field := func(data []byte) {
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], uint64(len(data)))
		_, _ = h.Write(n[:])
		_, _ = h.Write(data)
	}
	for _, args := range [][]string{{"status", "--porcelain=v1", "-z", "--untracked-files=all"}, {"diff", "--no-ext-diff", "--binary"}, {"diff", "--no-ext-diff", "--cached", "--binary"}} {
		out, err := git(args...)
		if err != nil {
			return "", err
		}
		field(out)
	}
	names, err := git("ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return "", err
	}
	for _, name := range bytes.Split(names, []byte{0}) {
		if len(name) == 0 {
			continue
		}
		field(name)
		path := filepath.Join(root, string(name))
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return "", err
			}
			field([]byte("link"))
			field([]byte(target))
		case info.Mode().IsRegular():
			f, err := os.Open(path)
			if err != nil {
				return "", err
			}
			fileHash := sha256.New()
			_, copyErr := io.Copy(fileHash, f)
			closeErr := f.Close()
			if copyErr != nil {
				return "", copyErr
			}
			if closeErr != nil {
				return "", closeErr
			}
			field([]byte("file"))
			field(fileHash.Sum(nil))
		default:
			return "", fmt.Errorf("unsupported untracked file type")
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
