package bootstrap

import (
	"encoding/json"
	"strings"
	"testing"

	t "github.com/scott-walker/mememory/internal/types"
)

func TestFormat_EmptyContext(testT *testing.T) {
	out := Format(Context{})
	if out != "" {
		testT.Errorf("Format with no memories should be empty, got %q", out)
	}
}

func TestFormat_GlobalOnly_ContainsSystemBlock(testT *testing.T) {
	out := Format(Context{
		GlobalMems: []t.Memory{
			{Content: "rule one", Type: t.TypeRule, Delivery: t.DeliveryBootstrap, Scope: t.ScopeGlobal},
		},
	})
	if !strings.Contains(out, "## System") {
		testT.Error("output missing ## System block")
	}
	if !strings.Contains(out, "rule one") {
		testT.Error("output missing memory content")
	}
}

func TestFormat_StatsBlockPresent(testT *testing.T) {
	out := Format(Context{
		Project: ProjectInfo{Name: "plexo", Source: ".mememory file"},
		GlobalMems: []t.Memory{
			{Content: "global rule", Type: t.TypeRule, Delivery: t.DeliveryBootstrap, Scope: t.ScopeGlobal},
		},
		ProjectMems: []t.Memory{
			{Content: "project rule", Type: t.TypeRule, Delivery: t.DeliveryBootstrap, Scope: t.ScopeProject, Project: "plexo"},
		},
	})

	if !strings.Contains(out, "## Bootstrap Stats") {
		testT.Error("output missing ## Bootstrap Stats block")
	}
	if !strings.Contains(out, "Project:   plexo") {
		testT.Error("stats missing project name")
	}
	if !strings.Contains(out, "source: .mememory file") {
		testT.Error("stats missing project source")
	}
	if !strings.Contains(out, "1 global + 1 project") {
		testT.Error("stats missing memory counts")
	}
	if !strings.Contains(out, "% of budget") {
		testT.Error("stats missing budget percent")
	}
}

func TestFormat_NoProject_ShowsGlobalsOnly(testT *testing.T) {
	out := Format(Context{
		GlobalMems: []t.Memory{
			{Content: "rule", Type: t.TypeRule, Delivery: t.DeliveryBootstrap, Scope: t.ScopeGlobal},
		},
	})
	if !strings.Contains(out, "(none — globals only)") {
		testT.Error("stats should report no project")
	}
}

func TestEstimateTokens(testT *testing.T) {
	cases := []struct {
		bytes int
		want  int
	}{
		{0, 0},
		{-5, 0},
		{7, 2}, // 7 / 3.5 = 2
		{35, 10},
		{350, 100},
	}
	for _, c := range cases {
		got := EstimateTokens(c.bytes)
		if got != c.want {
			testT.Errorf("EstimateTokens(%d) = %d, want %d", c.bytes, got, c.want)
		}
	}
}

func TestFormatThousands(testT *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{42, "42"},
		{1000, "1_000"},
		{30000, "30_000"},
		{1234567, "1_234_567"},
		{-1234, "-1_234"},
	}
	for _, c := range cases {
		got := formatThousands(c.in)
		if got != c.want {
			testT.Errorf("formatThousands(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestCheckBudget_WithinBudget(testT *testing.T) {
	mems := []t.Memory{
		{Content: "small", Type: t.TypeRule, Delivery: t.DeliveryBootstrap, Scope: t.ScopeGlobal},
	}
	if got := CheckBudget(mems); got != "" {
		testT.Errorf("CheckBudget for small set should be empty, got %q", got)
	}
}

func TestCheckBudget_OverBudget(testT *testing.T) {
	// Build a payload that comfortably exceeds 30K tokens (~ 105 KB at 3.5 bytes/token).
	huge := strings.Repeat("x", 200_000)
	mems := []t.Memory{
		{Content: huge, Type: t.TypeRule, Delivery: t.DeliveryBootstrap, Scope: t.ScopeGlobal},
	}
	got := CheckBudget(mems)
	if got == "" {
		testT.Error("CheckBudget for huge set should warn, got empty")
	}
	if !strings.Contains(got, "budget") {
		testT.Errorf("warning should mention budget, got %q", got)
	}
}

func TestCheckBudget_Empty(testT *testing.T) {
	if got := CheckBudget(nil); got != "" {
		testT.Errorf("CheckBudget for nil should be empty, got %q", got)
	}
}

func TestFormatHookJSON_EmptyContextReturnsEmpty(testT *testing.T) {
	out, err := FormatHookJSON(Context{})
	if err != nil {
		testT.Fatalf("FormatHookJSON error: %v", err)
	}
	if out != "" {
		testT.Errorf("empty context should produce empty string, got %q", out)
	}
}

func TestFormatHookJSON_WrapsMarkdownInEnvelope(testT *testing.T) {
	out, err := FormatHookJSON(Context{
		Project: ProjectInfo{Name: "plexo", Source: "test"},
		GlobalMems: []t.Memory{
			{Content: "global rule", Type: t.TypeRule, Delivery: t.DeliveryBootstrap, Scope: t.ScopeGlobal},
		},
	})
	if err != nil {
		testT.Fatalf("FormatHookJSON error: %v", err)
	}
	if out == "" {
		testT.Fatal("non-empty context should produce JSON, got empty")
	}

	var parsed struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		testT.Fatalf("output is not valid JSON: %v\npayload: %s", err, out)
	}

	if parsed.HookSpecificOutput.HookEventName != "SessionStart" {
		testT.Errorf("hookEventName = %q, want SessionStart", parsed.HookSpecificOutput.HookEventName)
	}
	if !strings.Contains(parsed.HookSpecificOutput.AdditionalContext, "## System") {
		testT.Error("additionalContext missing ## System block")
	}
	if !strings.Contains(parsed.HookSpecificOutput.AdditionalContext, "global rule") {
		testT.Error("additionalContext missing memory content")
	}
	if !strings.Contains(parsed.HookSpecificOutput.AdditionalContext, "## Bootstrap Stats") {
		testT.Error("additionalContext missing stats block")
	}
}

func TestFormatHookJSON_EscapesSpecialCharacters(testT *testing.T) {
	// Memory content with characters that must be escaped inside a JSON string:
	// double quotes, backslashes, newlines, and tabs. The output must remain
	// parseable JSON and the unmarshaled content must match exactly.
	content := "line1 \"quoted\"\nline2\t\\back"
	out, err := FormatHookJSON(Context{
		GlobalMems: []t.Memory{
			{Content: content, Type: t.TypeRule, Delivery: t.DeliveryBootstrap, Scope: t.ScopeGlobal},
		},
	})
	if err != nil {
		testT.Fatalf("FormatHookJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		testT.Fatalf("output is not valid JSON: %v\npayload: %s", err, out)
	}

	hso, ok := parsed["hookSpecificOutput"].(map[string]any)
	if !ok {
		testT.Fatal("hookSpecificOutput missing or wrong type")
	}
	ac, ok := hso["additionalContext"].(string)
	if !ok {
		testT.Fatal("additionalContext missing or wrong type")
	}
	if !strings.Contains(ac, content) {
		testT.Errorf("additionalContext did not round-trip special characters, got %q", ac)
	}
}

func TestFormat_OverflowWarning(testT *testing.T) {
	huge := strings.Repeat("x", 200_000)
	out := Format(Context{
		Project: ProjectInfo{Name: "plexo", Source: "test"},
		GlobalMems: []t.Memory{
			{Content: huge, Type: t.TypeRule, Delivery: t.DeliveryBootstrap, Scope: t.ScopeGlobal},
		},
	})
	if !strings.Contains(out, "WARNING") {
		testT.Error("overflow output should contain WARNING")
	}
	if !strings.Contains(out, "exceeds budget") {
		testT.Error("overflow output should mention exceeded budget")
	}
}
