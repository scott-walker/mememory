package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/scott-walker/mememory/internal/hooks"
)

// runRecallGate is the PreToolUse hook command. It blocks any tool call
// whose name does not start with "mcp__mememory__" while a recall-pending
// lock exists for the current session. The lock is created at SessionStart
// (by `mememory bootstrap --hook`) and cleared at PostToolUse on
// `mcp__mememory__recall` (by `mememory recall-ack`).
//
// Output protocol (Claude Code PreToolUse hook):
//   - exit 0 with no stdout = allow
//   - exit 0 with stdout JSON {"permissionDecision":"deny", ...} = deny
//
// The reason field is shown back to the agent, so we phrase it as an
// instruction it can act on rather than a passive error message.
func runRecallGate() error {
	input, err := hooks.ReadHookInputFromStdin()
	if err != nil {
		// Tolerant fallback: if we can't parse the hook payload, don't block.
		// The pinned-payload reinjection still carries the recall directive
		// in plain text — softer signal, but better than freezing the agent.
		return nil
	}

	if input.SessionID == "" {
		return nil
	}
	if !hooks.LockExists(input.SessionID) {
		return nil
	}
	if isMememoryTool(input.ToolName) {
		return nil
	}

	decision := map[string]any{
		"permissionDecision": "deny",
		"permissionDecisionReason": "Эта сессия требует первичного recall перед любыми другими операциями. " +
			"Вызови mcp__mememory__recall с запросом, релевантным текущему проекту, " +
			"чтобы загрузить полный контекст. После этого все остальные инструменты разблокируются.",
	}

	data, err := json.Marshal(decision)
	if err != nil {
		return fmt.Errorf("marshal decision: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func isMememoryTool(name string) bool {
	return strings.HasPrefix(name, "mcp__mememory__")
}
