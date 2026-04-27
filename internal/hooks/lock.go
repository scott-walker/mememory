package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// lockPrefix is the basename pattern for recall-pending lock files. The full
// path is built from os.TempDir() (cross-platform) + this prefix + session id.
const lockPrefix = "mememory-recall-pending-"

// LockPath returns the full path to the lock file for a given session id.
// Sessions without an id (empty string) get an empty path — callers should
// treat that as "no lock to manage".
func LockPath(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	// Sanitize: session_id is uuid-like, but be defensive against path-injection
	// in case the protocol changes upstream.
	safe := strings.ReplaceAll(sessionID, string(os.PathSeparator), "_")
	safe = strings.ReplaceAll(safe, "..", "_")
	return filepath.Join(os.TempDir(), lockPrefix+safe)
}

// CreateLock writes an empty file at LockPath(sessionID). Existing files are
// truncated — repeat calls within the same session are safe. Returns nil for
// empty session ids (silent no-op, matches LockPath's behaviour).
func CreateLock(sessionID string) error {
	path := LockPath(sessionID)
	if path == "" {
		return nil
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create lock: %w", err)
	}
	return f.Close()
}

// RemoveLock deletes the lock file for a session id. Missing files are not
// errors — that's the normal case after the first recall has cleared the lock.
func RemoveLock(sessionID string) error {
	path := LockPath(sessionID)
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove lock: %w", err)
	}
	return nil
}

// LockExists reports whether the lock file for a session id is currently on
// disk. False for empty session ids.
func LockExists(sessionID string) bool {
	path := LockPath(sessionID)
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// CleanStaleLocks removes lock files older than maxAge from os.TempDir().
// Used at SessionStart to garbage-collect locks left by crashed Claude Code
// sessions. Returns the number of files removed.
func CleanStaleLocks(maxAge time.Duration) (int, error) {
	dir := os.TempDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("read tmpdir: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), lockPrefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err == nil {
			removed++
		}
	}

	return removed, nil
}
