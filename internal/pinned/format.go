// Package pinned renders persisted pinned-delivery memories into the payload
// that the UserPromptSubmit hook injects on every agent turn. Unlike bootstrap
// (loaded once at session start), pinned content is reinjected continuously —
// so the payload is wrapped in a <system-reminder> tag and framed with a
// rotated imperative ("violation = task failure") to keep the rules acting as
// a checklist rather than fading into background.
//
// The output has two layers:
//   - System layer: meta-rules about working with mememory itself, plus rotated
//     framing texts. Hard-coded in internal/system_rules; not editable through
//     MCP tools.
//   - User layer: rules stored in Postgres with delivery=pinned, scope=global
//     or scope=project. Rendered as-is on stage 1 (LLM rotation of the user
//     layer is deferred to stage 2).
//
// Both layers are wrapped in a single <system-reminder> block. The block is
// emitted only when at least one user-layer memory is present — the system
// layer alone has nothing to anchor and would be reinjected on every turn even
// for installations that never configured pinned rules.
package pinned

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/scott-walker/mememory/internal/bootstrap"
	"github.com/scott-walker/mememory/internal/system_rules"
	t "github.com/scott-walker/mememory/internal/types"
)

// SoftBudgetTokens is the soft ceiling on the pinned-payload size. It is
// not a hard limit — the user layer is preserved verbatim regardless of
// length — but exceeding it produces an informational warning. The point
// is not to save context tokens (the agent runs in a 1M-token window) but
// to preserve the checklist effect: a payload growing past ~5K tokens
// becomes harder for the model to treat as a focused list.
const SoftBudgetTokens = 5_000

// BytesPerToken matches the ratio used by the bootstrap package — tuned for
// mixed Cyrillic prose and code. Per-tokenizer accuracy is out of scope.
const BytesPerToken = 3.5

// Context bundles everything Format needs to render a pinned payload. It is
// constructed by the CLI (after fetching memories from the admin API) or by
// the MCP resource handler (after listing memories from the engine).
type Context struct {
	Project     bootstrap.ProjectInfo
	GlobalMems  []t.Memory
	ProjectMems []t.Memory

	// Seed controls the rotation choice for the system layer. Zero means
	// "use time.Now().UnixNano()" — production behaviour. Tests pass a
	// fixed value to make the output deterministic.
	Seed int64
}

// Format renders a Context into the Markdown payload injected by the
// UserPromptSubmit hook. Returns an empty string when there are no
// user-layer memories — the caller should suppress hook output entirely
// in that case.
func Format(ctx Context) string {
	if len(ctx.GlobalMems) == 0 && len(ctx.ProjectMems) == 0 {
		return ""
	}

	seed := ctx.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	sel := system_rules.Select(seed)

	var b strings.Builder

	b.WriteString("<system-reminder>\n")
	b.WriteString(sel.FrameOpen)
	b.WriteString("\n\n")

	b.WriteString("Системные правила работы с памятью:\n")
	for _, mr := range sel.MetaRules {
		fmt.Fprintf(&b, "- %s\n", mr.Text)
	}
	b.WriteString("\n")

	if len(ctx.GlobalMems) > 0 {
		b.WriteString("Активные правила сессии:\n")
		for _, m := range ctx.GlobalMems {
			fmt.Fprintf(&b, "- %s\n", m.Content)
		}
		b.WriteString("\n")
	}

	if len(ctx.ProjectMems) > 0 {
		if ctx.Project.Name != "" {
			fmt.Fprintf(&b, "Правила проекта %s:\n", ctx.Project.Name)
		} else {
			b.WriteString("Правила проекта:\n")
		}
		for _, m := range ctx.ProjectMems {
			fmt.Fprintf(&b, "- %s\n", m.Content)
		}
		b.WriteString("\n")
	}

	b.WriteString(sel.FrameClose)
	b.WriteString("\n</system-reminder>\n")

	return b.String()
}

// FormatHookJSON wraps Format's output in the hookSpecificOutput envelope
// that Claude Code's UserPromptSubmit hook recognises. Runners that don't
// know the schema will print the JSON as plain text — noisy but not
// destructive. Returns an empty string when there is nothing to inject.
func FormatHookJSON(ctx Context) (string, error) {
	md := Format(ctx)
	if md == "" {
		return "", nil
	}

	payload := struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}{}
	payload.HookSpecificOutput.HookEventName = "UserPromptSubmit"
	payload.HookSpecificOutput.AdditionalContext = md

	out, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal hook payload: %w", err)
	}
	return string(out), nil
}

// EstimateTokens converts a byte count into an approximate token count using
// BytesPerToken. The estimate is rounded to the nearest integer.
func EstimateTokens(bytes int) int {
	if bytes <= 0 {
		return 0
	}
	return int(float64(bytes)/BytesPerToken + 0.5)
}

// CheckBudget returns a non-empty informational warning when the rendered
// pinned payload for the given memories would exceed SoftBudgetTokens.
// The check is non-blocking — callers store the memory regardless and just
// surface the warning to the user. Returns an empty string when within
// budget or when memories is empty.
//
// CheckBudget renders with a fixed seed so its measurement is reproducible
// — the size variation between rotations of the system layer is small
// enough that picking one is fine for budget purposes.
func CheckBudget(memories []t.Memory) string {
	if len(memories) == 0 {
		return ""
	}

	out := Format(Context{
		GlobalMems: memories,
		Seed:       1, // any non-zero, deterministic
	})
	tokens := EstimateTokens(len(out))
	if tokens <= SoftBudgetTokens {
		return ""
	}

	return fmt.Sprintf(
		"pinned payload is ~%s tokens (soft budget: %s). Pinned should stay tight to act as a checklist — consider trimming or moving non-critical rules to delivery=bootstrap.",
		formatThousands(tokens), formatThousands(SoftBudgetTokens),
	)
}

func formatThousands(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}

	digits := fmt.Sprintf("%d", n)
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}

	if negative {
		return "-" + b.String()
	}
	return b.String()
}
