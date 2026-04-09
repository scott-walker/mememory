// Package projectconfig parses the .mememory file that lives at a project root.
//
// The file is a JSON document that pins the canonical project name (and, in
// future schema versions, additional bootstrap/recall preferences) for any
// directory within the project tree. It is discovered via walk-up search from
// the current working directory, mirroring how git locates its repository root.
//
// Schema is versioned. v1 requires only the "version" and "project" fields.
// Reserved fields for future versions are documented in
// docs/config/mememory-file.md.
package projectconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FileName is the canonical name of the project config file. It lives at the
// project root and is discovered by walk-up search from any descendant
// directory.
const FileName = ".mememory"

// CurrentSchemaVersion is the highest schema version this build understands.
// Files with a higher version are read on a best-effort basis with a warning;
// unknown fields are ignored silently within a major version.
const CurrentSchemaVersion = 1

// File is the in-memory representation of a parsed .mememory document.
//
// Only fields defined in the current schema version are decoded explicitly.
// Unknown fields are dropped by encoding/json's default behavior, which gives
// us forward compatibility within the major version: a file written by a
// newer build of mememory remains readable by an older build, the older build
// just ignores fields it does not recognize.
type File struct {
	// Version is the schema version of this document. Required.
	Version int `json:"version"`

	// Project is the canonical project name used by mememory for scoping
	// memories. Required in v1.
	Project string `json:"project"`
}

// Found bundles a successfully located config file with the absolute path it
// was loaded from. The path is reported back to the user so they can see which
// file is governing the current session.
type Found struct {
	Path string
	File File
}

// ErrNotFound is returned by FindWalkUp when no .mememory file exists in the
// starting directory or any of its ancestors up to the filesystem root.
var ErrNotFound = errors.New("projectconfig: .mememory file not found")

// FindWalkUp searches for a .mememory file starting at startDir and walking
// upward through parent directories until one is found or the filesystem root
// is reached. The first match wins; ancestor files are not merged.
//
// Returns ErrNotFound if no file exists anywhere on the path. Returns a
// non-nil error wrapping the underlying parse failure if a file is found but
// invalid — callers must distinguish "not found" (silent fallback) from
// "found but broken" (loud failure).
func FindWalkUp(startDir string) (*Found, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, fmt.Errorf("projectconfig: resolve start dir: %w", err)
	}

	for {
		candidate := filepath.Join(dir, FileName)
		info, statErr := os.Stat(candidate)
		if statErr == nil && !info.IsDir() {
			file, loadErr := Load(candidate)
			if loadErr != nil {
				return nil, fmt.Errorf("projectconfig: load %s: %w", candidate, loadErr)
			}
			return &Found{Path: candidate, File: *file}, nil
		}
		if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
			return nil, fmt.Errorf("projectconfig: stat %s: %w", candidate, statErr)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding the file.
			return nil, ErrNotFound
		}
		dir = parent
	}
}

// Load reads and validates a .mememory file at the given absolute path. The
// caller is expected to have located the file (e.g. via FindWalkUp); Load does
// not perform walk-up search.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	if err := file.Validate(); err != nil {
		return nil, err
	}

	return &file, nil
}

// Validate checks the parsed file against the current schema rules. It is
// exported so tests and external callers can validate File values constructed
// in memory.
func (f *File) Validate() error {
	if f.Version == 0 {
		return errors.New("missing required field: version")
	}
	if f.Version < 1 {
		return fmt.Errorf("invalid version: %d (must be >= 1)", f.Version)
	}
	if f.Project == "" {
		return errors.New("missing required field: project")
	}
	return nil
}

// IsFutureVersion reports whether the file uses a schema version newer than
// what this build understands. Callers may choose to emit a warning and
// continue with best-effort parsing of the fields they recognize.
func (f *File) IsFutureVersion() bool {
	return f.Version > CurrentSchemaVersion
}
