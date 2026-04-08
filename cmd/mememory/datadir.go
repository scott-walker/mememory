package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ResolveDataDir returns the OS-standard data directory for mememory.
// Override via DATA_DIR environment variable.
// Creates the directory if it doesn't exist.
func ResolveDataDir() (string, error) {
	if env := os.Getenv("DATA_DIR"); env != "" {
		return ensureDir(env)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	var path string
	switch runtime.GOOS {
	case "linux":
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			path = filepath.Join(xdg, "mememory")
		} else {
			path = filepath.Join(home, ".local", "share", "mememory")
		}
	case "darwin":
		path = filepath.Join(home, "Library", "Application Support", "mememory")
	case "windows":
		if appdata := os.Getenv("LOCALAPPDATA"); appdata != "" {
			path = filepath.Join(appdata, "mememory")
		} else {
			path = filepath.Join(home, "AppData", "Local", "mememory")
		}
	default:
		path = filepath.Join(home, ".mememory")
	}

	return ensureDir(path)
}

func ensureDir(path string) (string, error) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", fmt.Errorf("create data directory %s: %w", path, err)
	}
	return path, nil
}
