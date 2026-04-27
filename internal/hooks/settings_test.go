package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchClaudeSettings_FreshFile_InstallCreatesAllHooks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	if err := PatchClaudeSettings(path, true); err != nil {
		t.Fatalf("install: %v", err)
	}

	settings := readJSON(t, path)
	hooks := mustMap(t, settings["hooks"])
	for _, mh := range managedHooks {
		eventList, ok := hooks[mh.Event].([]any)
		if !ok || len(eventList) == 0 {
			t.Errorf("event %q missing or empty after install", mh.Event)
			continue
		}
		if !entryHasManagedCommand(eventList[0], mh.CommandWord) {
			t.Errorf("event %q first entry doesn't contain managed command for %q", mh.Event, mh.CommandWord)
		}
	}
}

func TestPatchClaudeSettings_PreservesUnrelatedKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	initial := map[string]any{
		"language":  "Русский",
		"theme":     "light-ansi",
		"customNum": float64(42),
	}
	writeJSON(t, path, initial)

	if err := PatchClaudeSettings(path, true); err != nil {
		t.Fatalf("install: %v", err)
	}

	got := readJSON(t, path)
	if got["language"] != "Русский" {
		t.Errorf("language not preserved, got %v", got["language"])
	}
	if got["theme"] != "light-ansi" {
		t.Errorf("theme not preserved, got %v", got["theme"])
	}
	if got["customNum"] != float64(42) {
		t.Errorf("customNum not preserved, got %v", got["customNum"])
	}
}

func TestPatchClaudeSettings_PreservesExistingForeignHook(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	initial := map[string]any{
		"hooks": map[string]any{
			"SessionStart": []any{
				map[string]any{
					"matcher": "",
					"hooks": []any{
						map[string]any{"type": "command", "command": "some-other-tool init"},
					},
				},
			},
		},
	}
	writeJSON(t, path, initial)

	if err := PatchClaudeSettings(path, true); err != nil {
		t.Fatalf("install: %v", err)
	}

	got := readJSON(t, path)
	hooks := mustMap(t, got["hooks"])
	startList, ok := hooks["SessionStart"].([]any)
	if !ok || len(startList) < 2 {
		t.Fatalf("SessionStart should have foreign + ours, got %v", hooks["SessionStart"])
	}

	foundForeign := false
	foundOurs := false
	for _, entry := range startList {
		em, _ := entry.(map[string]any)
		inner, _ := em["hooks"].([]any)
		for _, h := range inner {
			hm, _ := h.(map[string]any)
			cmd, _ := hm["command"].(string)
			if cmd == "some-other-tool init" {
				foundForeign = true
			}
			if cmd == "mememory bootstrap --hook" {
				foundOurs = true
			}
		}
	}
	if !foundForeign {
		t.Error("foreign SessionStart hook lost")
	}
	if !foundOurs {
		t.Error("our SessionStart hook missing")
	}
}

func TestPatchClaudeSettings_InstallIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	if err := PatchClaudeSettings(path, true); err != nil {
		t.Fatalf("first install: %v", err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after first install: %v", err)
	}

	if err := PatchClaudeSettings(path, true); err != nil {
		t.Fatalf("second install: %v", err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after second install: %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("settings should be byte-identical after second install\n first: %s\nsecond: %s", first, second)
	}
}

func TestPatchClaudeSettings_PreservesCustomisedCommand(t *testing.T) {
	// User edited "mememory bootstrap --hook" to "mememory bootstrap --hook --url http://custom".
	// Install must NOT overwrite their customisation.
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	customised := map[string]any{
		"hooks": map[string]any{
			"SessionStart": []any{
				map[string]any{
					"matcher": "",
					"hooks": []any{
						map[string]any{"type": "command", "command": "mememory bootstrap --hook --url http://custom"},
					},
				},
			},
		},
	}
	writeJSON(t, path, customised)

	if err := PatchClaudeSettings(path, true); err != nil {
		t.Fatalf("install: %v", err)
	}

	got := readJSON(t, path)
	hooks := mustMap(t, got["hooks"])
	startList, _ := hooks["SessionStart"].([]any)
	if len(startList) != 1 {
		t.Fatalf("SessionStart should have 1 entry (customised one), got %d", len(startList))
	}
	em, _ := startList[0].(map[string]any)
	inner, _ := em["hooks"].([]any)
	hm, _ := inner[0].(map[string]any)
	cmd, _ := hm["command"].(string)
	if !strings.Contains(cmd, "--url http://custom") {
		t.Errorf("customisation lost, got command %q", cmd)
	}
}

func TestPatchClaudeSettings_UninstallRemovesAllManaged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	if err := PatchClaudeSettings(path, true); err != nil {
		t.Fatalf("install: %v", err)
	}
	if err := PatchClaudeSettings(path, false); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	got := readJSON(t, path)
	if hooks, ok := got["hooks"]; ok {
		t.Errorf("hooks key should be gone after full uninstall, got %v", hooks)
	}
}

func TestPatchClaudeSettings_UninstallKeepsForeignHooks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	initial := map[string]any{
		"hooks": map[string]any{
			"SessionStart": []any{
				map[string]any{
					"matcher": "",
					"hooks": []any{
						map[string]any{"type": "command", "command": "some-other-tool init"},
					},
				},
			},
		},
	}
	writeJSON(t, path, initial)

	_ = PatchClaudeSettings(path, true)
	_ = PatchClaudeSettings(path, false)

	got := readJSON(t, path)
	hooks := mustMap(t, got["hooks"])
	startList, ok := hooks["SessionStart"].([]any)
	if !ok || len(startList) != 1 {
		t.Fatalf("foreign hook should remain, got %v", hooks["SessionStart"])
	}
}

func TestPatchClaudeSettings_BackupCreatedOnInstall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	writeJSON(t, path, map[string]any{"language": "Русский"})

	if err := PatchClaudeSettings(path, true); err != nil {
		t.Fatalf("install: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	foundBackup := false
	for _, e := range entries {
		if strings.Contains(e.Name(), "settings.json.mememory-backup-") {
			foundBackup = true
			break
		}
	}
	if !foundBackup {
		t.Error("expected a .mememory-backup-* file in the dir")
	}
}

func TestPatchClaudeSettings_NoBackupIfNoFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	if err := PatchClaudeSettings(path, true); err != nil {
		t.Fatalf("install: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), "mememory-backup-") {
			t.Errorf("no backup expected when source file didn't exist, found %s", e.Name())
		}
	}
}

func TestCommandMatchesWord(t *testing.T) {
	cases := []struct {
		cmd  string
		word string
		want bool
	}{
		{"mememory bootstrap --hook", "bootstrap", true},
		{"mememory bootstrap", "bootstrap", true},
		{"/usr/local/bin/mememory bootstrap --hook", "bootstrap", true},
		{"mememory pinned", "bootstrap", false},
		{"some-other-tool init", "bootstrap", false},
		{"", "bootstrap", false},
		{"mememory", "bootstrap", false},
	}
	for _, c := range cases {
		if got := commandMatchesWord(c.cmd, c.word); got != c.want {
			t.Errorf("commandMatchesWord(%q, %q) = %v, want %v", c.cmd, c.word, got, c.want)
		}
	}
}

// --- helpers ---

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return got
}

func writeJSON(t *testing.T, path string, v map[string]any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func mustMap(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T (%v)", v, v)
	}
	return m
}
