---
name: Release
about: Release checklist for a new mememory version
title: "Release vX.Y.Z"
labels: release
assignees: scott-walker
---

## Release vX.Y.Z

### Phase 1 — Pre-release QA

- [ ] All CI checks pass on `main` (lint, vet, test, race)
- [ ] Run full test suite locally: `go test ./... -race -count=1`
- [ ] Verify Docker build succeeds: `docker build -f docker/Dockerfile .`
- [ ] Test CLI binary manually:
  - [ ] `mememory version` prints expected version
  - [ ] `mememory bootstrap --url http://localhost:4200` connects
  - [ ] `mememory status` reports health correctly
- [ ] Test mememory-server starts and responds on health endpoint
- [ ] Test mememory-admin serves web UI and API
- [ ] Branding check: `grep -r "claude-memory" .` returns zero matches
- [ ] Review CHANGELOG / prepare release notes draft
- [ ] Confirm no TODO/FIXME items block the release

### Phase 2 — Version Bump

- [ ] Update `Version` variable in `cmd/mememory/main.go`
- [ ] Update version references in `README.md` (install commands, badges)
- [ ] Update version in `docs/setup.md` if referenced
- [ ] Update version in `site/index.html` if referenced
- [ ] Commit: `git commit -m "chore: bump version to vX.Y.Z"`
- [ ] Tag and push: `git tag vX.Y.Z && git push origin vX.Y.Z`

### Phase 3 — Database & Data

- [ ] Verify no pending schema migrations
- [ ] If schema changed:
  - [ ] Document migration steps in release notes
  - [ ] Test migration from previous version's schema
  - [ ] Provide rollback instructions
- [ ] Verify `scripts/setup.sh` works on fresh database
- [ ] Confirm pgvector version compatibility documented

### Phase 4 — Build & Publish

- [ ] Tag push triggers release workflow
- [ ] Monitor GitHub Actions:
  - [ ] Test job passes
  - [ ] GoReleaser job completes (binaries, archives, checksums, Homebrew tap)
  - [ ] Docker job pushes to `ghcr.io/scott-walker/mememory`
- [ ] Verify GitHub Release page:
  - [ ] Release notes accurate
  - [ ] All platform archives attached
  - [ ] Checksum file present
- [ ] Verify Homebrew: `brew install scott-walker/tap/mememory`
- [ ] Verify Docker image:
  - [ ] `docker pull ghcr.io/scott-walker/mememory:vX.Y.Z`
  - [ ] `docker pull ghcr.io/scott-walker/mememory:latest`
  - [ ] Container starts, health endpoint responds

### Phase 5 — Documentation & Wiki

- [ ] Update `docs/setup.md` with new setup steps
- [ ] Update `docs/mcp-tools.md` if MCP tool signatures changed
- [ ] Update `docs/memory-model.md` if memory model changed
- [ ] Update `docs/architecture.md` if architecture changed
- [ ] Verify VitePress site builds and deploys (if applicable)
- [ ] Update README badges

### Phase 6 — Branding & Presentation

- [ ] Final branding audit: no "claude-memory" references
- [ ] All user-facing strings use "mememory" consistently
- [ ] CLI help text, error messages, log output correct
- [ ] Docker image labels/metadata correct
- [ ] GitHub repo description and topics current

### Phase 7 — Community & Visibility

- [ ] Write release announcement
- [ ] Update external documentation / integration guides
- [ ] If breaking changes: notify known users
- [ ] Tag release in project management tools

### Post-release Verification

- [ ] Install from Homebrew on clean machine
- [ ] Install from GitHub Release binary on Linux
- [ ] `mememory version` confirms new version
- [ ] Docker image end-to-end with docker-compose
- [ ] `mememory bootstrap` works against Docker-deployed instance
- [ ] Monitor GitHub Issues for 48 hours
