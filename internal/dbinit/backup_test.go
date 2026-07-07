package dbinit

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnsureLocalBackupAtTime_CreatesDailyBackupAndRetention(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shark-tasks.db")
	if err := os.WriteFile(dbPath, []byte("db-v1"), 0o644); err != nil {
		t.Fatalf("write db: %v", err)
	}
	if err := os.WriteFile(dbPath+"-wal", []byte("wal-v1"), 0o644); err != nil {
		t.Fatalf("write wal: %v", err)
	}

	policy := BackupPolicy{Enabled: true, BackupFiles: 2}
	day1 := time.Date(2026, 7, 6, 8, 0, 0, 0, time.UTC)
	day2 := day1.Add(24 * time.Hour)
	day3 := day2.Add(24 * time.Hour)

	if err := ensureLocalBackupAtTime(dbPath, policy, day1); err != nil {
		t.Fatalf("day1 backup: %v", err)
	}
	if err := ensureLocalBackupAtTime(dbPath, policy, day1.Add(2*time.Hour)); err != nil {
		t.Fatalf("same-day backup skip: %v", err)
	}

	day1Backup := filepath.Join(backupRoot(dbPath), "2026-07-06", "shark-tasks.db")
	if _, err := os.Stat(day1Backup); err != nil {
		t.Fatalf("expected day1 backup: %v", err)
	}

	if err := os.WriteFile(dbPath, []byte("db-v2"), 0o644); err != nil {
		t.Fatalf("update db for day2: %v", err)
	}
	if err := ensureLocalBackupAtTime(dbPath, policy, day2); err != nil {
		t.Fatalf("day2 backup: %v", err)
	}
	if err := os.WriteFile(dbPath, []byte("db-v3"), 0o644); err != nil {
		t.Fatalf("update db for day3: %v", err)
	}
	if err := ensureLocalBackupAtTime(dbPath, policy, day3); err != nil {
		t.Fatalf("day3 backup: %v", err)
	}

	if _, err := os.Stat(filepath.Join(backupRoot(dbPath), "2026-07-06")); !os.IsNotExist(err) {
		t.Fatalf("expected oldest backup directory to be pruned, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(backupRoot(dbPath), "2026-07-07", "shark-tasks.db")); err != nil {
		t.Fatalf("expected day2 backup to remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(backupRoot(dbPath), "2026-07-08", "shark-tasks.db")); err != nil {
		t.Fatalf("expected day3 backup to remain: %v", err)
	}
}

func TestEnsureLocalBackupAtTime_SkipsWhenDisabledOrMissingDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "shark-tasks.db")

	if err := ensureLocalBackupAtTime(dbPath, BackupPolicy{Enabled: false, BackupFiles: 7}, time.Now()); err != nil {
		t.Fatalf("disabled backups should not fail: %v", err)
	}
	if err := ensureLocalBackupAtTime(dbPath, BackupPolicy{Enabled: true, BackupFiles: 7}, time.Now()); err != nil {
		t.Fatalf("missing DB should be skipped: %v", err)
	}
}

func TestLoadBackupPolicy_DefaultsAndFields(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".sharkconfig.json")
	body := []byte(`{"backups":true,"backup_files":3}`)
	if err := os.WriteFile(configPath, body, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	policy, err := loadBackupPolicy(configPath)
	if err != nil {
		t.Fatalf("loadBackupPolicy: %v", err)
	}
	if !policy.Enabled {
		t.Fatal("expected backups enabled")
	}
	if policy.BackupFiles != 3 {
		t.Fatalf("backup_files = %d, want 3", policy.BackupFiles)
	}
}
