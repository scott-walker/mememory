package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// managedHook describes one of the four hooks `mememory install-hooks` writes
// into ~/.claude/settings.json. The fields are everything the patcher needs:
// which event to register under, what matcher pattern to use, the command word
// that identifies "this is our hook" during idempotency checks, and the full
// command string used when inserting a fresh entry.
type managedHook struct {
	Event       string
	Matcher     string
	CommandWord string
	FullCommand string
}

// managedHooks is the canonical list of hooks the installer manages. Order
// matters only for tests/output stability — Claude Code itself doesn't care.
var managedHooks = []managedHook{
	{Event: "SessionStart", Matcher: "", CommandWord: "bootstrap", FullCommand: "mememory bootstrap --hook"},
	{Event: "UserPromptSubmit", Matcher: "", CommandWord: "pinned", FullCommand: "mememory pinned --hook"},
	{Event: "PreToolUse", Matcher: "", CommandWord: "recall-gate", FullCommand: "mememory recall-gate"},
	{Event: "PostToolUse", Matcher: "mcp__mememory__recall", CommandWord: "recall-ack", FullCommand: "mememory recall-ack"},
}

// PatchClaudeSettings ensures (install=true) or removes (install=false) the
// four mememory hooks in a Claude Code settings.json file. Other settings —
// language, theme, custom hooks the user added — are preserved untouched.
//
// Idempotent: install leaves an existing mememory hook entry alone (so users
// who tweaked the command — e.g., added --url — keep their customisation).
// Uninstall removes ALL entries whose command starts with "mememory <word>"
// for our managed words, regardless of customisation.
//
// A backup at <path>.mememory-backup-<timestamp> is written before the file
// is modified. Missing source file is not an error — install creates a new
// settings.json from scratch in that case.
func PatchClaudeSettings(path string, install bool) error {
	settings, err := readSettings(path)
	if err != nil {
		return err
	}

	if err := backupSettings(path); err != nil {
		return fmt.Errorf("backup: %w", err)
	}

	if install {
		for _, mh := range managedHooks {
			ensureHook(settings, mh)
		}
	} else {
		for _, mh := range managedHooks {
			removeHook(settings, mh)
		}
		cleanupHooks(settings)
	}

	return writeSettings(path, settings)
}

func readSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, fmt.Errorf("read settings: %w", err)
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}
	if settings == nil {
		settings = map[string]any{}
	}
	return settings, nil
}

func writeSettings(path string, settings map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("ensure dir: %w", err)
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func backupSettings(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	backupPath := fmt.Sprintf("%s.mememory-backup-%s", path, time.Now().UTC().Format("20060102-150405"))
	return os.WriteFile(backupPath, data, 0o644)
}

func ensureHook(settings map[string]any, mh managedHook) {
	hooks := getOrCreateMap(settings, "hooks")
	eventList := getList(hooks, mh.Event)

	if findManagedEntry(eventList, mh.CommandWord) >= 0 {
		return
	}

	newEntry := map[string]any{
		"matcher": mh.Matcher,
		"hooks": []any{
			map[string]any{"type": "command", "command": mh.FullCommand},
		},
	}
	hooks[mh.Event] = append(eventList, newEntry)
}

func removeHook(settings map[string]any, mh managedHook) {
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return
	}
	eventList := getList(hooks, mh.Event)

	var filtered []any
	for _, entry := range eventList {
		if entryHasManagedCommand(entry, mh.CommandWord) {
			continue
		}
		filtered = append(filtered, entry)
	}

	if len(filtered) == 0 {
		delete(hooks, mh.Event)
	} else {
		hooks[mh.Event] = filtered
	}
}

// cleanupHooks drops the top-level "hooks" key entirely if it became empty
// after uninstallation. Avoids leaving an empty {"hooks": {}} stub.
func cleanupHooks(settings map[string]any) {
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		return
	}
	if len(hooks) == 0 {
		delete(settings, "hooks")
	}
}

// findManagedEntry returns the index of the first entry in eventList whose
// inner hooks contain a command for our managed word, or -1 if absent.
func findManagedEntry(eventList []any, word string) int {
	for i, entry := range eventList {
		if entryHasManagedCommand(entry, word) {
			return i
		}
	}
	return -1
}

func entryHasManagedCommand(entry any, word string) bool {
	entryMap, ok := entry.(map[string]any)
	if !ok {
		return false
	}
	innerHooks, ok := entryMap["hooks"].([]any)
	if !ok {
		return false
	}
	for _, h := range innerHooks {
		hMap, ok := h.(map[string]any)
		if !ok {
			continue
		}
		cmd, ok := hMap["command"].(string)
		if !ok {
			continue
		}
		if commandMatchesWord(cmd, word) {
			return true
		}
	}
	return false
}

// commandMatchesWord reports whether the shell command starts with
// "[/path/]mememory <word>". Matches both bare-name and absolute-path
// invocations, ignores trailing flags or arguments.
func commandMatchesWord(cmd, word string) bool {
	parts := strings.Fields(cmd)
	if len(parts) < 2 {
		return false
	}
	binary := filepath.Base(parts[0])
	return binary == "mememory" && parts[1] == word
}

func getOrCreateMap(parent map[string]any, key string) map[string]any {
	if existing, ok := parent[key].(map[string]any); ok {
		return existing
	}
	m := map[string]any{}
	parent[key] = m
	return m
}

func getList(parent map[string]any, key string) []any {
	if existing, ok := parent[key].([]any); ok {
		return existing
	}
	return nil
}
