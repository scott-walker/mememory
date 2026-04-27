package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLockPath_EmptySessionID(t *testing.T) {
	if got := LockPath(""); got != "" {
		t.Errorf("empty session id should give empty path, got %q", got)
	}
}

func TestLockPath_BuildsUnderTempDir(t *testing.T) {
	got := LockPath("abc-123")
	if !strings.HasPrefix(got, os.TempDir()) {
		t.Errorf("path %q should be under TempDir %q", got, os.TempDir())
	}
	if !strings.Contains(got, "mememory-recall-pending-abc-123") {
		t.Errorf("path %q should contain prefix and session id", got)
	}
}

func TestLockPath_SanitizesPathSeparator(t *testing.T) {
	got := LockPath("../malicious")
	if strings.Contains(got, "..") {
		t.Errorf("path %q should not contain path-traversal segment", got)
	}
}

func TestCreateLock_RemoveLock_LockExists(t *testing.T) {
	sid := "test-session-" + uniqueSuffix(t)
	defer func() { _ = RemoveLock(sid) }()

	if LockExists(sid) {
		t.Fatal("lock should not exist before creation")
	}
	if err := CreateLock(sid); err != nil {
		t.Fatalf("CreateLock: %v", err)
	}
	if !LockExists(sid) {
		t.Error("lock should exist after creation")
	}
	if err := RemoveLock(sid); err != nil {
		t.Fatalf("RemoveLock: %v", err)
	}
	if LockExists(sid) {
		t.Error("lock should not exist after removal")
	}
}

func TestCreateLock_EmptySessionID_NoOp(t *testing.T) {
	if err := CreateLock(""); err != nil {
		t.Errorf("CreateLock with empty id should be no-op, got %v", err)
	}
}

func TestRemoveLock_MissingFile_NoError(t *testing.T) {
	sid := "never-created-" + uniqueSuffix(t)
	if err := RemoveLock(sid); err != nil {
		t.Errorf("RemoveLock for missing file should not error, got %v", err)
	}
}

func TestRemoveLock_IdempotentAfterRemoval(t *testing.T) {
	sid := "test-idemp-" + uniqueSuffix(t)
	if err := CreateLock(sid); err != nil {
		t.Fatalf("CreateLock: %v", err)
	}
	if err := RemoveLock(sid); err != nil {
		t.Fatalf("first RemoveLock: %v", err)
	}
	if err := RemoveLock(sid); err != nil {
		t.Errorf("second RemoveLock should be no-op, got %v", err)
	}
}

func TestCleanStaleLocks_RemovesOldKeepsFresh(t *testing.T) {
	freshSid := "fresh-" + uniqueSuffix(t)
	staleSid := "stale-" + uniqueSuffix(t)
	if err := CreateLock(freshSid); err != nil {
		t.Fatalf("CreateLock fresh: %v", err)
	}
	defer func() { _ = RemoveLock(freshSid) }()
	if err := CreateLock(staleSid); err != nil {
		t.Fatalf("CreateLock stale: %v", err)
	}
	defer func() { _ = RemoveLock(staleSid) }()

	// Backdate the stale lock by 48h.
	stalePath := LockPath(staleSid)
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(stalePath, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	removed, err := CleanStaleLocks(24 * time.Hour)
	if err != nil {
		t.Fatalf("CleanStaleLocks: %v", err)
	}
	if removed < 1 {
		t.Errorf("expected at least 1 stale lock removed, got %d", removed)
	}
	if LockExists(staleSid) {
		t.Error("stale lock should be removed")
	}
	if !LockExists(freshSid) {
		t.Error("fresh lock should remain")
	}
}

// uniqueSuffix builds a per-test session-id suffix so parallel runs don't
// collide on shared TempDir.
func uniqueSuffix(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(filepath.Join(os.TempDir()), "mememory-test-uniq-*")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	name := filepath.Base(f.Name())
	_ = f.Close()
	_ = os.Remove(f.Name())
	return name
}
