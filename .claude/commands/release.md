Release mememory — full production release pipeline.

## Input

$ARGUMENTS — optional version override (e.g. "0.2.0" or "patch"/"minor"/"major"). If empty, version is determined automatically from the diff.

## Instructions

Execute the full release pipeline for the mememory project. Follow every step in order. Do not skip phases.

### Phase 0: Pre-release QA

1. Run `go vet ./...` — must be clean
2. Run `go build ./...` — must compile
3. Run `go test ./...` — all tests pass
4. Run `go test -race ./...` — no race conditions
5. Run `docker compose -f docker/docker-compose.yml -p mememory build` — Docker builds
6. Run `mememory status` — existing deployment healthy
7. Run `mememory bootstrap` — bootstrap works from current directory
8. Grep entire repo for "claude-memory" — must return zero matches. If found, fix before proceeding.

If any check fails — STOP and report.

### Phase 1: Version bump

1. Determine the new version:
   - If $ARGUMENTS is a semver string (e.g. "0.2.0"), use it directly
   - If $ARGUMENTS is "patch"/"minor"/"major", read current version and increment
   - If $ARGUMENTS is empty, auto-detect from diff:
     a. Get last tag: `git describe --tags --abbrev=0 2>/dev/null` (if no tags, current is 0.0.0)
     b. Get full diff: `git diff <last_tag>..HEAD --stat` and `git log --oneline <last_tag>..HEAD`
     c. Analyze changes and apply these rules:
        - **major** (X.0.0) if ANY of: MCP tool parameters changed/removed, database schema breaking change, Go module path changed, embedding dimension default changed, existing env vars renamed/removed
        - **minor** (0.X.0) if ANY of: new MCP tool or parameter added, new CLI command or flag, new embedding provider, new env var added, new API endpoint, significant new feature
        - **patch** (0.0.X) if ALL changes are: bug fixes, documentation updates, internal refactors, dependency updates, CI/CD changes, branding fixes, performance improvements
     d. Present the detected version to the user with reasoning before proceeding
2. Update the `Version` default in `cmd/mememory/main.go`
3. Update the version in `cmd/memory-server/main.go` (the `server.NewMCPServer("mememory", "X.Y.Z", ...)` call)
4. Update CHANGELOG.md:
   - Add a section for the new version with today's date
   - List changes from `git log --oneline $(git describe --tags --abbrev=0)..HEAD`
   - Categorize: Added, Changed, Fixed, Breaking
5. Run `go build ./...` again to verify

### Phase 2: Branding & Documentation audit

1. Grep the entire repo for old brand names ("claude-memory", "memory-cli"). Fix any found.
2. Verify all user-facing strings say "mememory" (not variations)
3. Check that docs/ pages reference correct version, commands, env vars
4. Check that README.md quick start is current
5. Check that site/index.html (landing page) is current

### Phase 3: Commit & Tag

1. Stage all changed files: `git add -A`
2. Create commit: "release: vX.Y.Z" with summary of changes
3. Create annotated tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`
4. Show the diff and tag for user confirmation before pushing

### Phase 4: Push & Publish

After user confirms:

1. `git push origin main`
2. `git push origin vX.Y.Z`
3. CI (GitHub Actions) handles:
   - GoReleaser: builds binaries, creates GitHub Release, updates Homebrew tap
   - Docker: builds and pushes image to ghcr.io
   - Docs: deploys VitePress site (if configured)
4. Monitor CI: `gh run list --limit 3`

### Phase 5: Post-release verification

1. Wait for CI to complete: `gh run watch`
2. Verify GitHub Release exists: `gh release view vX.Y.Z`
3. Verify `go install github.com/scott-walker/mememory@vX.Y.Z` works (dry run: check module proxy)
4. Rebuild local binary: `go build -o bin/mememory ./cmd/mememory && bin/mememory version`
5. Install locally: `cp bin/mememory ~/.local/bin/mememory`
6. Test: `mememory bootstrap`, `mememory status`

### Phase 6: Report

Print a release summary:
- Version released
- GitHub Release URL
- Number of changes
- Breaking changes (if any)
- Next steps (if any manual steps remain)
