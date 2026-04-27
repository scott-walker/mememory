# Pinned Rules & Forced Recall

`pinned` is the third memory delivery mode (alongside `bootstrap` and `on_demand`). Where bootstrap loads once at session start, pinned **reinjects on every agent turn** through the UserPromptSubmit hook, framed as a checklist the agent must verify before responding.

Pinned exists because rules loaded into bootstrap drift up the conversation context as the session grows and stop carrying the same weight after a few turns. Agents adapt to the noise, and rules that should be hard ("respond in Russian", "never use emoji", "never run `rm -rf` without confirmation") quietly degrade into background. Pinned solves that by re-presenting the rules every turn, wrapped in a `<system-reminder>` block with rotated framing so the model can't habituate to a single phrasing.

Forced recall is a related, but separate, mechanism that ensures the agent calls `mcp__mememory__recall` as the very first operation in a session. It uses a PreToolUse hook that physically blocks every other tool call until recall has run.

## How It Works

```
Agent session starts
    ↓
SessionStart hook → mememory bootstrap --hook
    │  • prints bootstrap markdown payload (as before)
    │  • creates a recall-pending lock file keyed by session_id
    ↓
On every user prompt:
    UserPromptSubmit hook → mememory pinned --hook
        • renders pinned-payload (system layer + global + project)
        • injects it via additionalContext in the JSON envelope
    ↓
On every tool invocation:
    PreToolUse hook → mememory recall-gate
        • if no lock → allow
        • if lock + tool starts with mcp__mememory__ → allow
        • otherwise → deny with a message instructing the agent to recall
    ↓
On mcp__mememory__recall completion:
    PostToolUse hook (matcher: mcp__mememory__recall) → mememory recall-ack
        • removes the lock — all tools unblocked for the rest of the session
```

The pinned payload itself contains a meta-rule reminding the agent that recall is mandatory at session start, so even when the PreToolUse hook is absent (custom installations, non-Claude-Code agents) the directive still arrives via plain text every turn. Soft signal for those clients, hard gate when hooks are wired.

## Pinned vs Bootstrap

| | bootstrap | pinned |
|---|---|---|
| When loaded | Once at session start | Every agent turn |
| Hook | SessionStart | UserPromptSubmit |
| Use for | Facts, project context, framing — things the agent should know | Hard rules, behavioural imperatives — things the agent must verify against |
| Wrapped in | `## System` markdown header | `<system-reminder>` block with rotated framing imperative |
| Soft budget | 30,000 tokens | 5,000 tokens (loose) |
| Source | Postgres `delivery=bootstrap` | Postgres `delivery=pinned` |

The two layers are complementary, not redundant. A typical setup has 10–30 bootstrap memories (project facts, user identity, stack details) and 5–10 pinned memories (hard rules that must hold on every turn).

## Output Format

Pinned output is a single `<system-reminder>` block wrapping three layers: a rotated framing imperative, a system meta-rules section managed by mememory itself, and the user's pinned memories grouped by scope.

```markdown
<system-reminder>
Чек-лист перед ответом: пройдись по списку правил ниже и проверь применимость каждого пункта к своему ответу.

Системные правила работы с памятью:
- mememory — единственный источник долговременной памяти. Игнорируй встроенные механизмы Claude Code (auto-memory, MEMORY.md, ~/.claude/*/memory/).
- На первом сообщении сессии recall — обязательная первая операция. Без него работа над задачей запрещена.
- Память отражает состояние на момент записи. Если факт из памяти противоречит текущему состоянию кода — доверяй коду, не памяти.
- Нарушение pinned-правила = провал задачи, не "почти получилось". Без градаций.
- Bootstrap, загруженный в начале сессии — рабочий справочник. Сверяй свои предположения с ним.

Активные правила сессии:
- Respond only in Russian.
- Never use emoji in code, UI, comments, or commit messages.

Правила проекта mememory:
- Any database migration must be data-preserving by design.

Перед действием: подтверди, что ни одно из правил выше не нарушается.
</system-reminder>
```

The opening and closing imperatives, plus each system meta-rule, are randomly chosen from a pool of equivalent formulations on every render. This rotation defends against agent adaptation: a payload that arrives in a slightly different shape every turn is harder to filter out than one that repeats verbatim.

User-layer pinned memories are emitted as-is — no LLM rotation in the current release. The payload always wraps in `<system-reminder>` so models that treat that tag with elevated weight (Claude Code among them) take the contents seriously.

## Token Budget

`SoftBudgetTokens` (`internal/pinned/format.go`) is **5,000 tokens** — a loose ceiling, not a hard limit. The constant is denominated in tokens, estimated from byte length using the same `BytesPerToken=3.5` ratio bootstrap uses.

The budget exists not to save context (modern context windows are large) but to preserve the **checklist effect**. A 50-rule pinned payload dilutes attention regardless of how much context is available — pinned must stay tight to act as a checklist rather than a manual.

When `remember(delivery="pinned", ...)` is called and the resulting set would render past `SoftBudgetTokens`, the response is prefixed with a warning suggesting the user trim the payload or move non-critical rules to `delivery=bootstrap`. The memory is stored either way — the warning is informational.

## Hook Configuration

The simplest path is `mememory install-hooks`, which patches `~/.claude/settings.json` with all four hooks. It's idempotent, preserves existing settings and foreign hooks, and writes a backup at `~/.claude/settings.json.mememory-backup-<timestamp>` before any change.

```bash
mememory install-hooks            # add the four hooks
mememory install-hooks --uninstall # remove them, keep everything else
mememory install-hooks --path /custom/settings.json  # alternate target
```

After install, `~/.claude/settings.json` contains:

```json
{
  "hooks": {
    "SessionStart": [
      {"matcher": "", "hooks": [{"type": "command", "command": "mememory bootstrap --hook"}]}
    ],
    "UserPromptSubmit": [
      {"matcher": "", "hooks": [{"type": "command", "command": "mememory pinned --hook"}]}
    ],
    "PreToolUse": [
      {"matcher": "", "hooks": [{"type": "command", "command": "mememory recall-gate"}]}
    ],
    "PostToolUse": [
      {"matcher": "mcp__mememory__recall", "hooks": [{"type": "command", "command": "mememory recall-ack"}]}
    ]
  }
}
```

If an entry with a `mememory <command>` already exists (for example, you've added flags like `--url`), the installer leaves it alone — your customisation persists across re-runs. Only fresh installs add new entries.

::: warning OpenAI Codex CLI
The current installer targets Claude Code only. Codex hooks live in `~/.codex/hooks.json` with a slightly different schema and are scheduled for the next release. For now, Codex users can replicate the four hooks manually following the structure above (Codex's hook event names are the same).
:::

## Forced Recall

The lock-file mechanism behind PreToolUse blocking:

1. SessionStart writes `${TMPDIR}/mememory-recall-pending-${session_id}` (an empty file).
2. PreToolUse on any tool checks: does the lock exist? If yes, and the tool name doesn't start with `mcp__mememory__`, return `permissionDecision: "deny"` with an instructional reason.
3. PostToolUse on `mcp__mememory__recall` (gated by Claude Code's matcher) removes the lock. From this point on, all tools work normally for the rest of the session.

The lock is keyed on the Claude Code-issued `session_id`, so multiple concurrent sessions don't interfere. Stale locks (older than 24 hours, e.g. left by crashed sessions) are garbage-collected on the next SessionStart.

If a Claude Code release ever breaks the PreToolUse hook contract, the system degrades gracefully: `recall-gate` defaults to "allow" on parse failure, and the pinned payload still carries the recall directive in plain text.

## CLI

For inspection without hooks:

```bash
mememory pinned                       # raw markdown (manual inspection)
mememory pinned --hook                # JSON envelope for hook runners
mememory pinned --project mememory    # include project-scoped pinned
mememory pinned --url http://...      # custom Admin API URL
```

`--hook` output shape:

```json
{
  "hookSpecificOutput": {
    "hookEventName": "UserPromptSubmit",
    "additionalContext": "<rendered pinned payload>"
  }
}
```

Project resolution mirrors `mememory bootstrap`: `--project` flag → `.mememory` file (walk-up from cwd) → `git rev-parse --show-toplevel` basename → `basename(cwd)`.

If the Admin API is unreachable, `mememory pinned` exits silently with no output — same fail-soft contract as bootstrap, so a hook never blocks the agent's turn.

## MCP Resources (Alternative)

For MCP clients that read resources at connection time:

| URI | Content |
|---|---|
| `mememory://pinned` | Global pinned memories only |
| `mememory://pinned/{project}` | Global + project-scoped pinned memories |

These return the same payload as `mememory pinned --project ...` but through the MCP channel. Resource support varies by client — the UserPromptSubmit hook approach is more reliable.

## Admin UI

The admin UI (`http://localhost:4200`) has a **Pinned Preview** page that renders the exact payload your agent receives for any project. The page also shows token estimates and counts of pinned memories at each scope. Useful when designing the pinned set or debugging unexpected agent behaviour.

The standard memories page now supports `delivery=pinned` as a filter, and the New Memory form lets you create pinned entries directly.

## Filtering

`mememory pinned` loads only memories with `delivery=pinned` from the scope hierarchy. Everything else — bootstrap, on_demand, expired — is excluded. Project pinned memories never leak across projects: a session in project A only sees global pinned + A's project pinned.

Up to 100 memories per scope level are fetched. If you ever hit that ceiling, your pinned set is far past the soft budget — the soft warning will have been firing well before then.

## Silent Failure

If the Admin API is unreachable, `mememory pinned` exits silently with no output (exit code 0). The hook's empty stdout is treated as "nothing to inject" by Claude Code, so the agent's turn proceeds without pinned context. This matches the bootstrap behaviour and ensures a misconfigured or stopped admin service never blocks agent operation.
