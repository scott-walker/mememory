package projectconfig

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidV1(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	writeFile(t, path, `{"version":1,"project":"plexo"}`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.Version != 1 {
		t.Errorf("Version = %d, want 1", got.Version)
	}
	if got.Project != "plexo" {
		t.Errorf("Project = %q, want %q", got.Project, "plexo")
	}
}

func TestLoad_MissingVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	writeFile(t, path, `{"project":"plexo"}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing version, got nil")
	}
}

func TestLoad_MissingProject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	writeFile(t, path, `{"version":1}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing project, got nil")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	writeFile(t, path, `not json`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoad_FutureVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	writeFile(t, path, `{"version":99,"project":"plexo","unknown":"field"}`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load with future version should succeed (forward-compat): %v", err)
	}
	if !got.IsFutureVersion() {
		t.Error("IsFutureVersion() = false, want true")
	}
	if got.Project != "plexo" {
		t.Errorf("Project = %q, want %q", got.Project, "plexo")
	}
}

func TestLoad_UnknownFieldsIgnored(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	writeFile(t, path, `{"version":1,"project":"plexo","_comment":"hi","bootstrap":{"budget_tokens":50000}}`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.Project != "plexo" {
		t.Errorf("Project = %q, want %q", got.Project, "plexo")
	}
}

func TestFindWalkUp_FileInStartDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, FileName), `{"version":1,"project":"plexo"}`)

	found, err := FindWalkUp(dir)
	if err != nil {
		t.Fatalf("FindWalkUp returned error: %v", err)
	}
	if found.File.Project != "plexo" {
		t.Errorf("Project = %q, want %q", found.File.Project, "plexo")
	}
	if filepath.Base(found.Path) != FileName {
		t.Errorf("Path basename = %q, want %q", filepath.Base(found.Path), FileName)
	}
}

func TestFindWalkUp_FileInAncestor(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, FileName), `{"version":1,"project":"plexo"}`)

	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	found, err := FindWalkUp(deep)
	if err != nil {
		t.Fatalf("FindWalkUp returned error: %v", err)
	}
	if found.File.Project != "plexo" {
		t.Errorf("Project = %q, want %q", found.File.Project, "plexo")
	}
}

func TestFindWalkUp_FirstWins(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, FileName), `{"version":1,"project":"outer"}`)

	inner := filepath.Join(root, "inner")
	if err := os.MkdirAll(inner, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeFile(t, filepath.Join(inner, FileName), `{"version":1,"project":"inner"}`)

	found, err := FindWalkUp(inner)
	if err != nil {
		t.Fatalf("FindWalkUp returned error: %v", err)
	}
	if found.File.Project != "inner" {
		t.Errorf("Project = %q, want %q (first match should win)", found.File.Project, "inner")
	}
}

func TestFindWalkUp_NotFound(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	_, err := FindWalkUp(deep)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestFindWalkUp_BrokenFileSurfaces(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, FileName), `{"version":}`)

	_, err := FindWalkUp(dir)
	if err == nil {
		t.Fatal("expected error for broken file, got nil")
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("broken file should not be reported as ErrNotFound")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
