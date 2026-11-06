//go:build unix

package fileops

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

const fileopsTestUmaskEnv = "SHARK_FILEOPS_TEST_UMASK"

func TestAtomicWriteRespectsUmask(t *testing.T) {
	maskText, child := os.LookupEnv(fileopsTestUmaskEnv)
	if !child {
		for _, mask := range []string{"022", "077"} {
			cmd := exec.Command(os.Args[0], "-test.run=^TestAtomicWriteRespectsUmask$")
			cmd.Env = environmentWithout(fileopsTestUmaskEnv)
			cmd.Env = append(cmd.Env, fileopsTestUmaskEnv+"="+mask)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "umask %s subprocess output: %s", mask, output)
		}
		return
	}

	mask, err := strconv.ParseUint(maskText, 8, 32)
	require.NoError(t, err)
	syscall.Umask(int(mask))

	tmpDir := t.TempDir()
	writer := NewEntityFileWriter()
	_, err = writer.WriteEntityFile(WriteOptions{
		Content:         []byte("umask check"),
		ProjectRoot:     tmpDir,
		FilePath:        "umask.md",
		EntityType:      "task",
		UseAtomicWrite:  true,
		CreateIfMissing: true,
	})
	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(tmpDir, "umask.md"))
	require.NoError(t, err)
	expected := os.FileMode(0644) &^ os.FileMode(mask)
	require.Equal(t, expected, info.Mode().Perm(), "umask %s must limit requested file permissions", maskText)
}

func environmentWithout(name string) []string {
	var env []string
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, name+"=") {
			env = append(env, entry)
		}
	}
	return env
}
