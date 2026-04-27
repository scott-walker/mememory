package pinned

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/scott-walker/mememory/internal/bootstrap"
	t "github.com/scott-walker/mememory/internal/types"
)

func TestFormat_EmptyContext(testT *testing.T) {
	out := Format(Context{})
	if out != "" {
		testT.Errorf("Format with no memories should be empty, got %q", out)
	}
}

func TestFormat_GlobalOnly_ContainsSystemReminderEnvelope(testT *testing.T) {
	out := Format(Context{
		GlobalMems: []t.Memory{
			{Content: "rule one", Type: t.TypeRule, Delivery: t.DeliveryPinned, Scope: t.ScopeGlobal},
		},
		Seed: 1,
	})
	if !strings.HasPrefix(out, "<system-reminder>\n") {
		testT.Error("output should start with <system-reminder> tag")
	}
	if !strings.HasSuffix(out, "</system-reminder>\n") {
		testT.Error("output should end with closing </system-reminder> tag")
	}
	if !strings.Contains(out, "Системные правила работы с памятью:") {
		testT.Error("output missing system meta-rules section header")
	}
	if !strings.Contains(out, "Активные правила сессии:") {
		testT.Error("output missing global pinned section header")
	}
	if !strings.Contains(out, "rule one") {
		testT.Error("output missing memory content")
	}
}

func TestFormat_ProjectOnly_ShowsProjectSection(testT *testing.T) {
	out := Format(Context{
		Project: bootstrap.ProjectInfo{Name: "voif", Source: "test"},
		ProjectMems: []t.Memory{
			{Content: "project rule", Type: t.TypeRule, Delivery: t.DeliveryPinned, Scope: t.ScopeProject, Project: "voif"},
		},
		Seed: 1,
	})
	if !strings.Contains(out, "Правила проекта voif:") {
		testT.Error("output missing project section header with project name")
	}
	if !strings.Contains(out, "project rule") {
		testT.Error("output missing project memory content")
	}
	if strings.Contains(out, "Активные правила сессии:") {
		testT.Error("output should not show global section when GlobalMems is empty")
	}
}

func TestFormat_BothScopes_GlobalBeforeProject(testT *testing.T) {
	out := Format(Context{
		Project: bootstrap.ProjectInfo{Name: "voif", Source: "test"},
		GlobalMems: []t.Memory{
			{Content: "global rule", Type: t.TypeRule, Delivery: t.DeliveryPinned, Scope: t.ScopeGlobal},
		},
		ProjectMems: []t.Memory{
			{Content: "project rule", Type: t.TypeRule, Delivery: t.DeliveryPinned, Scope: t.ScopeProject, Project: "voif"},
		},
		Seed: 1,
	})

	globalIdx := strings.Index(out, "global rule")
	projectIdx := strings.Index(out, "project rule")
	if globalIdx == -1 || projectIdx == -1 {
		testT.Fatalf("missing rules in output: globalIdx=%d projectIdx=%d", globalIdx, projectIdx)
	}
	if globalIdx >= projectIdx {
		testT.Errorf("global rule should appear before project rule (general → specific), but globalIdx=%d projectIdx=%d", globalIdx, projectIdx)
	}
}

func TestFormat_DeterministicWithSeed(testT *testing.T) {
	mems := []t.Memory{
		{Content: "rule", Type: t.TypeRule, Delivery: t.DeliveryPinned, Scope: t.ScopeGlobal},
	}
	a := Format(Context{GlobalMems: mems, Seed: 42})
	b := Format(Context{GlobalMems: mems, Seed: 42})
	if a != b {
		testT.Error("same seed should produce identical output")
	}
}

func TestFormat_SystemLayerRotates(testT *testing.T) {
	mems := []t.Memory{
		{Content: "rule", Type: t.TypeRule, Delivery: t.DeliveryPinned, Scope: t.ScopeGlobal},
	}
	first := Format(Context{GlobalMems: mems, Seed: 1})

	allSame := true
	for seed := int64(2); seed < 100; seed++ {
		next := Format(Context{GlobalMems: mems, Seed: seed})
		if next != first {
			allSame = false
			break
		}
	}
	if allSame {
		testT.Error("system layer should rotate across seeds, but 100 seeds produced identical output")
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

func TestFormatHookJSON_HookEventName(testT *testing.T) {
	out, err := FormatHookJSON(Context{
		GlobalMems: []t.Memory{
			{Content: "rule", Type: t.TypeRule, Delivery: t.DeliveryPinned, Scope: t.ScopeGlobal},
		},
		Seed: 1,
	})
	if err != nil {
		testT.Fatalf("FormatHookJSON error: %v", err)
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

	if parsed.HookSpecificOutput.HookEventName != "UserPromptSubmit" {
		testT.Errorf("hookEventName = %q, want UserPromptSubmit", parsed.HookSpecificOutput.HookEventName)
	}
	if !strings.Contains(parsed.HookSpecificOutput.AdditionalContext, "<system-reminder>") {
		testT.Error("additionalContext should contain system-reminder envelope")
	}
	if !strings.Contains(parsed.HookSpecificOutput.AdditionalContext, "rule") {
		testT.Error("additionalContext missing memory content")
	}
}

func TestFormatHookJSON_EscapesSpecialCharacters(testT *testing.T) {
	content := "line1 \"quoted\"\nline2\t\\back"
	out, err := FormatHookJSON(Context{
		GlobalMems: []t.Memory{
			{Content: content, Type: t.TypeRule, Delivery: t.DeliveryPinned, Scope: t.ScopeGlobal},
		},
		Seed: 1,
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

func TestEstimateTokens(testT *testing.T) {
	cases := []struct {
		bytes int
		want  int
	}{
		{0, 0},
		{-5, 0},
		{7, 2},
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

func TestCheckBudget_Empty(testT *testing.T) {
	if got := CheckBudget(nil); got != "" {
		testT.Errorf("CheckBudget for nil should be empty, got %q", got)
	}
}

func TestCheckBudget_WithinBudget(testT *testing.T) {
	mems := []t.Memory{
		{Content: "small rule", Type: t.TypeRule, Delivery: t.DeliveryPinned, Scope: t.ScopeGlobal},
	}
	if got := CheckBudget(mems); got != "" {
		testT.Errorf("CheckBudget for small set should be empty, got %q", got)
	}
}

func TestCheckBudget_OverBudget(testT *testing.T) {
	// SoftBudgetTokens = 5_000 ≈ 17_500 bytes. 50_000 byte content well exceeds it.
	huge := strings.Repeat("x", 50_000)
	mems := []t.Memory{
		{Content: huge, Type: t.TypeRule, Delivery: t.DeliveryPinned, Scope: t.ScopeGlobal},
	}
	got := CheckBudget(mems)
	if got == "" {
		testT.Error("CheckBudget for huge set should warn, got empty")
	}
	if !strings.Contains(got, "soft budget") {
		testT.Errorf("warning should mention soft budget, got %q", got)
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
		{5000, "5_000"},
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
