package hooks

import (
	"strings"
	"testing"
)

func TestReadHookInput_Empty(t *testing.T) {
	got, err := ReadHookInput(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != (HookInput{}) {
		t.Errorf("empty input should produce zero HookInput, got %+v", got)
	}
}

func TestReadHookInput_FullSchema(t *testing.T) {
	src := `{
		"session_id": "abc-123",
		"hook_event_name": "PreToolUse",
		"tool_name": "mcp__mememory__recall",
		"tool_use_id": "use-456",
		"cwd": "/home/x",
		"source": "startup"
	}`
	got, err := ReadHookInput(strings.NewReader(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := HookInput{
		SessionID:     "abc-123",
		HookEventName: "PreToolUse",
		ToolName:      "mcp__mememory__recall",
		ToolUseID:     "use-456",
		Cwd:           "/home/x",
		Source:        "startup",
	}
	if got != want {
		t.Errorf("mismatch\n got=%+v\nwant=%+v", got, want)
	}
}

func TestReadHookInput_UnknownFieldsIgnored(t *testing.T) {
	src := `{"session_id": "x", "future_field_we_dont_care_about": 42}`
	got, err := ReadHookInput(strings.NewReader(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SessionID != "x" {
		t.Errorf("SessionID = %q, want %q", got.SessionID, "x")
	}
}

func TestReadHookInput_InvalidJSON(t *testing.T) {
	_, err := ReadHookInput(strings.NewReader("{not valid"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
