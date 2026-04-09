# `.mememory` File Specification

The `.mememory` file pins the canonical project name (and, in future schema
versions, additional bootstrap and recall preferences) for any directory
inside a project tree. It lives at the project root and is discovered via
walk-up search from the current working directory, mirroring how `git`
locates its repository root.

This file is the single source of truth for "what project am I in?" when the
mememory CLI is invoked from anywhere inside the project. It removes the
ambiguity of relying on git basename or `cwd` for projects whose directory
name does not match their canonical mememory name (for example, a project
called `plexo` that lives at `/home/scott/dev/remide/projects`).

## Location and Discovery

- File name: `.mememory` (no extension, dotfile convention).
- Location: project root, alongside `.git`, `package.json`, etc.
- Discovery: `mememory bootstrap` walks up from `cwd` toward `/`, taking the
  **first** `.mememory` file it finds. Ancestor files are not merged.
- One file per project tree. Do not nest multiple `.mememory` files inside a
  single project unless you intentionally want a sub-tree to identify as a
  different project.

## Format

UTF-8 encoded JSON, no BOM. Comments are not supported by JSON; if you need
to leave a note for human readers, use a field prefixed with `_` such as
`_comment` — the parser ignores unknown fields within a major schema version.

## Schema v1

```json
{
  "version": 1,
  "project": "plexo"
}
```

### Required fields (v1)

| Field     | Type   | Description                                        |
|-----------|--------|----------------------------------------------------|
| `version` | int    | Schema version. Currently `1`.                     |
| `project` | string | Canonical project name used by mememory scoping.  |

### Validation rules

- `version` is required. A missing or zero version is a hard error.
- `version` must be a positive integer.
- `project` is required and must be a non-empty string.
- Unknown fields are silently ignored within the same major version, which
  gives forward compatibility: a file written by a newer build of mememory
  remains readable by an older build.
- A `version` greater than the highest version this build understands
  produces a warning but is still parsed on a best-effort basis.

## Project Resolution Priority

When `mememory bootstrap` runs, it determines the project name through this
priority chain. The first source that yields a non-empty name wins; the
chosen source is reported in the bootstrap stats block so the user can see
exactly which rule was applied.

1. **`--project` CLI flag.** Explicit override, always wins.
2. **`.mememory` file.** Discovered via walk-up from `cwd`.
3. **`git rev-parse --show-toplevel` basename.** When inside a git repo.
4. **`basename(cwd)`.** Last-resort fallback.

## Reserved Fields (Roadmap)

The following field paths are **reserved** for future schema versions and
must not be used to mean anything else. Files that include them today will
parse cleanly under v1 (unknown-field tolerance), but their behavior is
undefined until the corresponding feature ships.

```json
{
  "version": 1,
  "project": "plexo",

  "bootstrap": {
    "budget_tokens": 30000,
    "include_tags": ["critical"],
    "exclude_tags": ["experimental"]
  },

  "recall": {
    "auto_query": "plexo architecture context",
    "auto_limit": 15
  },

  "remember": {
    "default_tags": ["plexo"],
    "default_scope": "project"
  },

  "agent": {
    "profile": "claude-opus-4-6-1m"
  }
}
```

| Path                       | Intended use                                          |
|----------------------------|-------------------------------------------------------|
| `bootstrap.budget_tokens`  | Override the default 30K-token bootstrap budget.      |
| `bootstrap.include_tags`   | Restrict bootstrap loading to memories with any tag.  |
| `bootstrap.exclude_tags`   | Skip memories carrying any listed tag.                |
| `recall.auto_query`        | Query string for an automatic SessionStart recall.    |
| `recall.auto_limit`        | Result count cap for the auto-recall.                 |
| `remember.default_tags`    | Tags applied to every memory stored from this tree.   |
| `remember.default_scope`   | Default scope for `remember` calls (global/project).  |
| `agent.profile`            | Hint about which agent/model the project targets.     |

## Versioning Policy

- **Major versions break compatibility.** A file written for v2 may not
  parse correctly under a v1 build, and vice versa. The `version` field
  identifies the major version.
- **Minor schema additions inside a major version are non-breaking.** New
  optional fields can be added without bumping `version`. Unknown fields
  are ignored, so older builds keep working.
- **Removing or repurposing a field requires a major version bump.** This
  applies even to fields documented as reserved.
- A future `mememory config migrate` command will handle upgrades between
  major versions when the need arises.

## Example: Single-Project Layout

```text
~/dev/myproject/
├── .git/
├── .mememory          ← {"version":1,"project":"myproject-canonical-name"}
├── README.md
└── src/
    └── ...
```

Running `mememory bootstrap` from anywhere under `~/dev/myproject/` resolves
the project to `myproject-canonical-name`.

## Example: Mono-Repo with Per-Sub-Tree Override

```text
~/dev/monorepo/
├── .mememory          ← {"version":1,"project":"monorepo-shared"}
├── apps/
│   ├── frontend/
│   │   └── .mememory  ← {"version":1,"project":"monorepo-frontend"}
│   └── backend/
│       └── .mememory  ← {"version":1,"project":"monorepo-backend"}
└── packages/
    └── ...            ← falls through to monorepo-shared
```

Walk-up gives each sub-tree its own canonical project. Anything that does not
sit under a `.mememory`-bearing sub-tree falls back to the root file.
