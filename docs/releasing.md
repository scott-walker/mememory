# Release Checklist

Complete all phases sequentially. Do not skip phases even if they seem unnecessary for a given release.

## Phase 1 — Pre-release QA

- [ ] All CI checks pass on `main` (lint, vet, test, race)
- [ ] Run full test suite locally: `go test ./... -race -count=1`
- [ ] Verify Docker build succeeds: `docker build -f docker/Dockerfile .`
- [ ] Test CLI binary manually:
  - `mememory version` prints expected version
  - `mememory bootstrap --url http://localhost:4200` connects to a running instance
  - `mememory status` reports service health correctly
- [ ] Test mememory-server starts and responds on health endpoint
- [ ] Test mememory-admin serves web UI and API
- [ ] Run branding check: `grep -r "claude-memory" . --include="*.go" --include="*.md" --include="*.ts" --include="*.html"` — must return zero matches
- [ ] Review CHANGELOG or prepare release notes draft
- [ ] Confirm no TODO/FIXME items block the release

## Phase 2 — Version Bump

- [ ] Update `Version` variable in `cmd/mememory/main.go`
- [ ] Update version references in `README.md` if present (install commands, badges)
- [ ] Update version in `docs/setup.md` if referenced
- [ ] Update version in `site/index.html` if referenced
- [ ] Commit version bump: `git commit -m "chore: bump version to vX.Y.Z"`
- [ ] Create and push tag: `git tag vX.Y.Z && git push origin vX.Y.Z`

## Phase 3 — Database & Data

- [ ] Verify no pending schema migrations are required
- [ ] If schema changed since last release:
  - [ ] Document migration steps in release notes
  - [ ] Test migration from previous version's schema to current
  - [ ] Provide rollback instructions
- [ ] Verify `scripts/setup.sh` still works on a fresh database
- [ ] Confirm pgvector extension version compatibility is documented

## Phase 4 — Build & Publish

- [ ] Verify tag pushed triggers the release workflow
- [ ] Monitor GitHub Actions release workflow:
  - [ ] Test job passes
  - [ ] GoReleaser job completes (binaries, archives, checksums, Homebrew tap)
  - [ ] Docker job pushes image to `ghcr.io/scott-walker/mememory`
- [ ] Verify GitHub Release page:
  - [ ] Release notes are accurate
  - [ ] All platform archives are attached (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)
  - [ ] Checksum file is present
- [ ] Verify Docker image:
  - `docker pull ghcr.io/scott-walker/mememory:vX.Y.Z`
  - `docker pull ghcr.io/scott-walker/mememory:latest`
  - Container starts and health endpoint responds

## Phase 5 — Documentation & Wiki

- [ ] Update `docs/setup.md` with any new setup steps
- [ ] Update `docs/mcp-tools.md` if MCP tool signatures changed
- [ ] Update `docs/memory-model.md` if memory model changed
- [ ] Update `docs/architecture.md` if architecture changed
- [ ] If VitePress site exists under `site/`, verify it builds and deploys
- [ ] Update README badges (version, CI status)

## Phase 6 — Branding & Presentation

- [ ] Final branding audit: no references to "claude-memory" anywhere in the repo
- [ ] Verify all user-facing strings use "mememory" consistently
- [ ] Check CLI help text, error messages, and log output for correct naming
- [ ] Verify Docker image labels and metadata use correct project name
- [ ] Verify GitHub repo description and topics are current

## Phase 7 — Community & Visibility

- [ ] Write release announcement (GitHub Discussions or relevant channels)
- [ ] Update any external documentation or integration guides
- [ ] If breaking changes: notify known users/integrators
- [ ] Tag the release in any project management tools

## Post-release Verification

- [ ] Install from GitHub Release binary on Linux
- [ ] Run `mememory version` — confirms new version string
- [ ] Pull and run Docker image end-to-end with docker-compose
- [ ] Verify `mememory bootstrap` works against a Docker-deployed instance
- [ ] Monitor GitHub Issues for immediate bug reports (48 hours)
