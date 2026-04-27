// Package hooks provides shared utilities for the Claude Code hook commands
// implemented in cmd/mememory: stdin payload parsing, lock-file management
// for the forced-recall mechanism, and the ~/.claude/settings.json patcher
// that the `mememory install-hooks` command uses.
package hooks

import (
	"encoding/json"
	"io"
	"os"
)

// HookInput is the subset of fields we read from a hook stdin JSON payload.
// Claude Code sends different schemas for SessionStart / UserPromptSubmit /
// PreToolUse / PostToolUse — but the fields we care about have stable names
// across events, so a single tolerant struct handles all of them.
//
// Fields that don't apply to a given event remain zero-valued — that's the
// caller's signal that they weren't sent.
type HookInput struct {
	SessionID     string `json:"session_id"`
	HookEventName string `json:"hook_event_name"`
	ToolName      string `json:"tool_name"`
	ToolUseID     string `json:"tool_use_id"`
	Cwd           string `json:"cwd"`
	Source        string `json:"source"`
}

// ReadHookInput parses a hook stdin payload from r. Empty input or unknown
// fields are not errors — we get zero-valued fields the caller can branch on.
func ReadHookInput(r io.Reader) (HookInput, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return HookInput{}, err
	}
	if len(data) == 0 {
		return HookInput{}, nil
	}
	var input HookInput
	if err := json.Unmarshal(data, &input); err != nil {
		return HookInput{}, err
	}
	return input, nil
}

// ReadHookInputFromStdin is a convenience wrapper that reads from os.Stdin
// only when stdin is piped. When the binary is run from a TTY (manual
// invocation), it returns an empty HookInput without blocking on Read.
func ReadHookInputFromStdin() (HookInput, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return HookInput{}, nil
	}
	// CharDevice mode means stdin is a terminal — no piped JSON, return empty.
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return HookInput{}, nil
	}
	return ReadHookInput(os.Stdin)
}
