package runner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
)

type bugCandidateScope struct {
	ProjectRoot      string             `json:"project_root"`
	RuntimeProject   string             `json:"runtime_project"`
	BaseCommit       string             `json:"base_commit"`
	CandidateCommit  string             `json:"candidate_commit"`
	CandidateTree    string             `json:"candidate_tree"`
	DiffSHA256       string             `json:"diff_sha256"`
	RepairPlanSHA256 string             `json:"repair_plan_sha256"`
	ChangedFiles     []string           `json:"changed_files"`
	ChangedFileCount int                `json:"changed_file_count"`
	Source           *bugSourceSnapshot `json:"source_snapshot"`
}

type bugSourceSnapshot struct {
	ProjectRoot           string   `json:"project_root"`
	SourceRoot            string   `json:"source_root"`
	RuntimeProject        string   `json:"runtime_project"`
	SourceCommit          string   `json:"source_commit"`
	CandidateCommit       string   `json:"candidate_commit"`
	CandidateTree         string   `json:"candidate_tree"`
	SelectedPaths         []string `json:"selected_paths"`
	SelectedIndexSHA256   string   `json:"selected_index_sha256"`
	SourceInventorySHA256 string   `json:"source_inventory_sha256"`
}

func (s *bugSourceSnapshot) UnmarshalJSON(data []byte) error {
	type snapshotFields bugSourceSnapshot
	var value snapshotFields
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	*s = bugSourceSnapshot(value)
	return nil
}

// verifyBugCandidate checks the reviewed tree against the current source, including
// selected uncommitted files when the review used an isolated snapshot.
func verifyBugCandidate(ctx context.Context, root string, data []byte) error {
	gitProject, relativeProject, err := candidateRepository(ctx, root)
	if err != nil {
		return err
	}
	root = gitProject
	var scope bugCandidateScope
	if err := json.Unmarshal(data, &scope); err != nil {
		return fmt.Errorf("decode review scope: %w", err)
	}
	if scope.RuntimeProject == "" { // Legacy captures used the repository root.
		scope.RuntimeProject = "."
	}
	if scope.RuntimeProject != filepath.ToSlash(relativeProject) {
		return fmt.Errorf("review Shark project location mismatch")
	}
	git := func(env []string, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "core.fsmonitor=false", "-C", root}, args...)...)
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
	return validateBugSnapshot(ctx, root, scope, git, run)
}

func candidateRepository(ctx context.Context, root string) (string, string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve repository path: %w", err)
	}
	gitRoot, err := exec.CommandContext(ctx, "git", "-c", "core.fsmonitor=false", "-C", root, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", "", fmt.Errorf("find candidate repository: %w", err)
	}
	gitProject, err := filepath.EvalSymlinks(strings.TrimSpace(string(gitRoot)))
	if err != nil {
		return "", "", fmt.Errorf("resolve candidate repository: %w", err)
	}
	sharkProject, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", "", fmt.Errorf("resolve Shark project: %w", err)
	}
	relativeProject, err := filepath.Rel(gitProject, sharkProject)
	if err != nil {
		return "", "", fmt.Errorf("find Shark project within repository: %w", err)
	}

	return gitProject, relativeProject, nil
}

type bugGitRunner func(...string) ([]byte, error)
type bugGitWithEnv func([]string, ...string) ([]byte, error)

func validateBugCandidateManifest(scope bugCandidateScope, run bugGitRunner) error {
	if err := validateBugGitIdentity(scope, run); err != nil {
		return err
	}
	return validateBugDiffManifest(scope, run)
}
func validateBugGitIdentity(scope bugCandidateScope, run bugGitRunner) error {
	if !validGitHash(scope.BaseCommit) || !validGitHash(scope.CandidateCommit) || !validGitHash(scope.CandidateTree) {
		return fmt.Errorf("invalid candidate identity")
	}
	if _, err := run("cat-file", "-e", scope.CandidateCommit+"^{commit}"); err != nil {
		return fmt.Errorf("candidate commit is invalid: %w", err)
	}
	if _, err := run("merge-base", "--is-ancestor", scope.BaseCommit, scope.CandidateCommit); err != nil {
		return fmt.Errorf("review baseline is not a candidate ancestor: %w", err)
	}
	tree, err := run("rev-parse", scope.CandidateCommit+"^{tree}")
	if err != nil || strings.TrimSpace(string(tree)) != scope.CandidateTree {
		return fmt.Errorf("candidate tree mismatch")
	}
	return nil
}
func validGitHash(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
func validateBugDiffManifest(scope bugCandidateScope, run bugGitRunner) error {
	diff, err := run("diff", "--no-ext-diff", "--binary", scope.BaseCommit, scope.CandidateCommit)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(diff)
	if hex.EncodeToString(hash[:]) != scope.DiffSHA256 || len(diff) == 0 {
		return fmt.Errorf("candidate diff mismatch")
	}
	return validateBugChangedFiles(scope, run)
}

func validateBugChangedFiles(scope bugCandidateScope, run bugGitRunner) error {
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
func validateCleanBugCandidate(root string, scope bugCandidateScope, run bugGitRunner) error {
	if err := requireNoSnapshotMarker(root, scope.CandidateCommit, run); err != nil {
		return err
	}
	if err := sameBugProject(root, scope.ProjectRoot); err != nil {
		return err
	}
	return validateCleanBugHead(scope, run)
}
func requireNoSnapshotMarker(root, commit string, run bugGitRunner) error {
	marker, err := run("rev-parse", "--git-path", "shark-review-source.json")
	if err != nil {
		return err
	}
	path := strings.TrimSpace(string(marker))
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("snapshot source metadata missing")
	} else if !os.IsNotExist(err) {
		return err
	}
	subject, err := run("log", "-1", "--format=%s", commit)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(subject)) == "Shark isolated review snapshot v1" {
		return fmt.Errorf("snapshot source metadata missing")
	}
	return nil
}
func sameBugProject(a, b string) error {
	first, err := filepath.EvalSymlinks(a)
	if err != nil {
		return err
	}
	second, err := filepath.EvalSymlinks(b)
	if err != nil || first != second {
		return fmt.Errorf("review project mismatch")
	}
	return nil
}
func normalizedRuntimeProject(project string) string {
	if project == "" {
		return "."
	}
	return project
}

func cleanReviewStatus(runtimeProject string, run bugGitRunner) ([]byte, error) {
	prefix, err := reviewRuntimePrefix(runtimeProject)
	if err != nil {
		return nil, err
	}
	data, err := run("status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	return filterReviewStatus(data, prefix)
}

func validateCleanBugHead(scope bugCandidateScope, run bugGitRunner) error {
	head, err := run("rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(string(head)) != scope.CandidateCommit {
		return fmt.Errorf("candidate commit is stale")
	}
	status, err := cleanReviewStatus(scope.RuntimeProject, run)
	if err != nil || len(status) != 0 {
		return fmt.Errorf("candidate worktree changed")
	}
	return nil
}
func validateBugSnapshot(ctx context.Context, root string, scope bugCandidateScope, git bugGitWithEnv, run bugGitRunner) error {
	if err := validateRetainedSnapshot(ctx, root, scope); err != nil {
		return err
	}
	if err := validateSnapshotSource(root, scope, run); err != nil {
		return err
	}
	return reconstructSnapshotTree(scope, git)
}
func validateSnapshotSource(root string, scope bugCandidateScope, run bugGitRunner) error {
	source := scope.Source
	if err := sameBugProject(root, source.SourceRoot); err != nil {
		return fmt.Errorf("snapshot source mismatch: %w", err)
	}
	if err := validateSnapshotSelection(scope); err != nil {
		return err
	}
	head, err := run("rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(string(head)) != source.SourceCommit {
		return fmt.Errorf("snapshot source commit changed")
	}
	return validateSnapshotInventory(root, source, run)
}
func validateSnapshotSelection(scope bugCandidateScope) error {
	source := scope.Source
	if normalizedRuntimeProject(source.RuntimeProject) != scope.RuntimeProject || source.CandidateCommit != scope.CandidateCommit || source.CandidateTree != scope.CandidateTree || len(source.SelectedPaths) == 0 {
		return fmt.Errorf("snapshot identity mismatch")
	}
	prefix, err := reviewRuntimePrefix(scope.RuntimeProject)
	if err != nil {
		return err
	}
	for _, path := range source.SelectedPaths {
		if !filepath.IsLocal(path) || path == "." || selectionOverlapsRuntime(filepath.ToSlash(path), prefix) {
			return fmt.Errorf("invalid selected path")
		}
	}
	return nil
}

func validateSnapshotInventory(root string, source *bugSourceSnapshot, run bugGitRunner) error {
	inventory, err := bugSourceInventory(root, normalizedRuntimeProject(source.RuntimeProject), run)
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
	hash := sha256.Sum256(staged)
	if hex.EncodeToString(hash[:]) != source.SelectedIndexSHA256 {
		return fmt.Errorf("source index changed")
	}
	return nil
}
func reconstructSnapshotTree(scope bugCandidateScope, git bugGitWithEnv) (resultErr error) {
	temp, err := os.MkdirTemp("", "shark-review-index-")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(temp); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("remove temporary review index: %w", err))
		}
	}()
	env := append(os.Environ(), "GIT_INDEX_FILE="+filepath.Join(temp, "index"))
	if _, err = git(env, "read-tree", scope.Source.SourceCommit); err != nil {
		return err
	}
	args := append([]string{"--literal-pathspecs", "add", "-A", "--"}, scope.Source.SelectedPaths...)
	if _, err = git(env, args...); err != nil {
		return err
	}
	actual, err := git(env, "write-tree")
	if err != nil || strings.TrimSpace(string(actual)) != scope.CandidateTree {
		return fmt.Errorf("source candidate changed")
	}
	return nil
}

type snapshotGitRunner func(string, ...string) ([]byte, error)

// validateRetainedSnapshot requires the reviewed snapshot to remain clean and available through QA.
func validateRetainedSnapshot(ctx context.Context, sourceRoot string, scope bugCandidateScope) error {
	run := func(root string, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "core.fsmonitor=false", "-C", root}, args...)...)
		return cmd.Output()
	}
	if err := sameBugProject(scope.Source.ProjectRoot, scope.ProjectRoot); err != nil {
		return fmt.Errorf("snapshot project metadata mismatch: %w", err)
	}
	if err := validateSnapshotRepository(sourceRoot, scope.ProjectRoot, run); err != nil {
		return err
	}
	if err := validateRetainedWorktree(scope, run); err != nil {
		return err
	}
	return validateSnapshotMarker(scope, run)
}
func validateRetainedWorktree(scope bugCandidateScope, run snapshotGitRunner) error {
	head, err := run(scope.ProjectRoot, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(string(head)) != scope.CandidateCommit {
		return fmt.Errorf("snapshot commit changed or unavailable")
	}
	tree, err := run(scope.ProjectRoot, "rev-parse", "HEAD^{tree}")
	if err != nil || strings.TrimSpace(string(tree)) != scope.CandidateTree {
		return fmt.Errorf("snapshot tree changed")
	}
	status, err := cleanReviewStatus(scope.RuntimeProject, func(args ...string) ([]byte, error) { return run(scope.ProjectRoot, args...) })
	if err != nil || len(status) != 0 {
		return fmt.Errorf("snapshot worktree changed")
	}
	return nil
}
func validateSnapshotRepository(source, snapshot string, run snapshotGitRunner) error {
	common := func(root string) (string, error) {
		path, err := run(root, "rev-parse", "--git-common-dir")
		if err != nil {
			return "", err
		}
		value := strings.TrimSpace(string(path))
		if !filepath.IsAbs(value) {
			value = filepath.Join(root, value)
		}
		return filepath.EvalSymlinks(value)
	}
	original, err := common(source)
	if err != nil {
		return err
	}
	retained, err := common(snapshot)
	if err != nil || original != retained {
		return fmt.Errorf("snapshot repository mismatch")
	}
	return nil
}
func validateSnapshotMarker(scope bugCandidateScope, run snapshotGitRunner) error {
	gitDir, err := run(scope.ProjectRoot, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return err
	}
	raw, err := ReviewReadArtifact(filepath.Join(strings.TrimSpace(string(gitDir)), "shark-review-source.json"), 2*1024*1024)
	if err != nil {
		return fmt.Errorf("snapshot provenance marker: %w", err)
	}
	var marker bugSourceSnapshot
	if err := json.Unmarshal(raw, &marker); err != nil || !reflect.DeepEqual(marker, *scope.Source) {
		return fmt.Errorf("snapshot provenance marker mismatch")
	}
	return nil
}

func bugSourceInventory(root, runtimeProject string, git func(...string) ([]byte, error)) (string, error) {
	h := sha256.New()
	field := func(data []byte) {
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], uint64(len(data)))
		// hash.Hash.Write is documented never to return an error.
		_, _ = h.Write(n[:])
		_, _ = h.Write(data)
	}
	prefix, err := reviewRuntimePrefix(runtimeProject)
	if err != nil {
		return "", err
	}
	status, err := cleanReviewStatus(runtimeProject, git)
	if err != nil {
		return "", err
	}
	field(status)
	for _, args := range [][]string{{"diff", "--no-ext-diff", "--binary"}, {"diff", "--no-ext-diff", "--cached", "--binary"}} {
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
	if err := hashUntrackedInventory(root, prefix, names, field); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashUntrackedInventory(root, prefix string, names []byte, field func([]byte)) error {
	confined, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	// Closing a read-only directory handle cannot affect inventory integrity.
	defer func() { _ = confined.Close() }()
	for _, name := range bytes.Split(names, []byte{0}) {
		if len(name) == 0 || strings.HasPrefix(string(name), prefix) {
			continue
		}
		field(name)
		kind, content, err := bugSourceInventoryEntry(confined, string(name))
		if err != nil {
			return err
		}
		field([]byte(kind))
		field(content)
	}
	return nil
}

func bugSourceInventoryEntry(root *os.Root, name string) (string, []byte, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return "", nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return bugInventoryLink(root, name, info)
	}
	if !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("unsupported untracked file type")
	}
	file, err := bugOpenVerifiedInventoryFile(root, name, info)
	if err != nil {
		return "", nil, err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil {
		return "", nil, copyErr
	}
	if closeErr != nil {
		return "", nil, closeErr
	}
	return "file", hash.Sum(nil), nil
}

func bugOpenVerifiedInventoryFile(root *os.Root, name string, original os.FileInfo) (*os.File, error) {
	file, err := root.OpenFile(name, os.O_RDONLY|bugInventoryOpenFlags, 0)
	if err != nil {
		return nil, err
	}
	opened, statErr := file.Stat()
	if statErr != nil || !opened.Mode().IsRegular() || !os.SameFile(original, opened) {
		_ = file.Close() // Read-only close does not affect the rejected entry.
		return nil, fmt.Errorf("untracked file changed while opening")
	}
	current, err := root.Lstat(name)
	if err != nil || !current.Mode().IsRegular() || !os.SameFile(original, current) {
		_ = file.Close() // Read-only close does not affect the rejected entry.
		return nil, fmt.Errorf("untracked file changed while opening")
	}
	return file, nil
}

func bugInventoryLink(root *os.Root, name string, original os.FileInfo) (string, []byte, error) {
	target, err := root.Readlink(name)
	if err != nil {
		return "", nil, err
	}
	current, err := root.Lstat(name)
	if err != nil || !os.SameFile(original, current) {
		return "", nil, fmt.Errorf("untracked link changed while reading")
	}
	return "link", []byte(target), nil
}
