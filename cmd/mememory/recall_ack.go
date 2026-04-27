package main

import "github.com/scott-walker/mememory/internal/hooks"

// runRecallAck is the PostToolUse hook command, scoped via settings.json
// matcher to fire only after `mcp__mememory__recall`. It removes the
// recall-pending lock for the session, unblocking the recall-gate.
//
// Errors are intentionally swallowed: a missing lock is normal (the user
// may have called recall a second time in the same session), and a stale
// filesystem error shouldn't break the agent's flow.
func runRecallAck() error {
	input, err := hooks.ReadHookInputFromStdin()
	if err != nil || input.SessionID == "" {
		return nil
	}
	_ = hooks.RemoveLock(input.SessionID)
	return nil
}
