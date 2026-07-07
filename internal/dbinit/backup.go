package dbinit

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/jwwelbor/shark-task-manager/internal/config"
)

const defaultBackupFiles = 7

// BackupPolicy controls automatic local SQLite backups before startup migrations.
type BackupPolicy struct {
	Enabled     bool
	BackupFiles int
}

func loadBackupPolicy(configPath string) (BackupPolicy, error) {
	policy := BackupPolicy{BackupFiles: defaultBackupFiles}

	cfg, err := config.NewManager(configPath).Load()
	if err != nil {
		return policy, nil
	}
	if cfg == nil || cfg.RawData == nil {
		return policy, nil
	}

	if enabled, ok := cfg.RawData["backups"].(bool); ok {
		policy.Enabled = enabled
	}
	if raw, ok := cfg.RawData["backup_files"].(float64); ok {
		policy.BackupFiles = int(raw)
	}
	if raw, ok := cfg.RawData["backup_files"].(string); ok {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil {
			policy.BackupFiles = parsed
		}
	}
	if policy.BackupFiles <= 0 {
		policy.BackupFiles = defaultBackupFiles
	}

	return policy, nil
}

func ensureLocalBackup(dbPath string, policy BackupPolicy) error {
	return ensureLocalBackupAtTime(dbPath, policy, time.Now())
}

func ensureLocalBackupAtTime(dbPath string, policy BackupPolicy, now time.Time) error {
	if !policy.Enabled {
		return nil
	}

	info, err := os.Stat(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat database: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("database path is a directory: %s", dbPath)
	}

	dateKey := now.Format("2006-01-02")
	todayDir := filepath.Join(backupRoot(dbPath), dateKey)
	if _, err := os.Stat(filepath.Join(todayDir, filepath.Base(dbPath))); err == nil {
		return pruneOldBackups(dbPath, policy.BackupFiles)
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("check existing backup: %w", err)
	}

	if err := os.MkdirAll(todayDir, 0o755); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}

	for _, src := range backupSourceFiles(dbPath) {
		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("stat backup source %s: %w", src, err)
		}
		dst := filepath.Join(todayDir, filepath.Base(src))
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copy %s: %w", src, err)
		}
	}

	if err := pruneOldBackups(dbPath, policy.BackupFiles); err != nil {
		return err
	}

	return nil
}

func backupRoot(dbPath string) string {
	return filepath.Join(filepath.Dir(dbPath), ".shark-backups", filepath.Base(dbPath))
}

func backupSourceFiles(dbPath string) []string {
	return []string{dbPath, dbPath + "-wal", dbPath + "-shm"}
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	info, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		_ = os.Remove(dst)
		return err
	}
	if err := dstFile.Sync(); err != nil {
		_ = os.Remove(dst)
		return err
	}
	return nil
}

func pruneOldBackups(dbPath string, keep int) error {
	root := backupRoot(dbPath)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read backup directory: %w", err)
	}

	dirs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}
	sort.Strings(dirs)
	if len(dirs) <= keep {
		return nil
	}

	for _, stale := range dirs[:len(dirs)-keep] {
		if err := os.RemoveAll(filepath.Join(root, stale)); err != nil {
			return fmt.Errorf("remove stale backup %s: %w", stale, err)
		}
	}
	return nil
}
