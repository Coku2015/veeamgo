# Progress Report — Stage 1-3A (2025-10-12)

## Summary
Stage 1 (Foundations), Stage 2 (Infrastructure Management), and Stage 3A (Read-only Visibility) are feature-complete and verified against the lab environment. The CLI now supports automatic authentication/refresh from config files, kubectl-style noun/verb parity, raw JSON passthrough for every command, and a consistent table layout aligned with the latest UI design. Cross-platform release artifacts (macOS, Linux, Windows — amd64 & arm64) are produced for every build cycle.

## Delivered Highlights
- **Authentication & Sessions**
  - `login` / `logout` / `session get` commands operational on macOS, Linux, and Windows.
  - Sessions and saved passwords now use AES-GCM encrypted files under `~/.veeamgo/`, enabling uniform behaviour across macOS/Linux/Windows (no external keyring dependency).
  - CLI auto-creates `~/.veeamgo/config` on first run; encryption keys live next to the config for zero-touch bootstrap on new hosts.
  - Sample config at `docs/examples/veeamgo.config.example.yaml` enables zero-touch usage.
- **Server Insights**
  - `server get info` / `server get time` emit redesigned single-record views with timezone detection fixes.
- **Infrastructure Inventory**
  - Managed servers: list, describe, rescan (single/all) with optional `--wait` session monitoring.
  - Repositories: list, describe, rescan (single/all) with human-friendly Yes/No columns.
  - Inventory browsers: `veeamgo inventory servers|objects|physical` tap the stable POST endpoints with pagination and optional JSON filters.
- **Stage 3A Read-only Coverage**
  - Jobs, sessions, backups, restore points/replicas, and proxy inventories expose full read-only flows with raw API JSON passthrough whenever `--output json` is used.
  - The smoke harness consumes JSON output to drive describe/log commands while respecting the new verb-first CLI layout.
  - Security analyzer schedule/send-results reporting and licensing surfaces (`get license`, `get license sockets|instances|capacity`) deliver audit-ready summaries with friendlier table headers.
- **Stage 3B UX Polish**
  - Verb-first command wrappers (`veeamgo get|describe <resource>`) anchor the CLI around consistent automation-friendly verbs.
  - `restorepoint`/`replica` commands now require explicit job context, output job-aware tables, and remove redundant ID-heavy columns; legacy `backup get|describe`, `restore mount`, `task`, `credential`, and malware/security event commands were retired.
  - Repository, license, and job listings focus on operator-ready columns, with dedicated job-type flags mapping directly to each `EJobType` enumerated in the OpenAPI specification.
- **Release Tooling**
  - `dist/` holds binaries for macOS (amd64, arm64), Linux (amd64, arm64), Windows (amd64).
  - Output renderer revamped to match the target UI specification.

## Verification & QA
- Manual validation run against `vbrsav13.backupnext.home` using `.env.test` coverage for every Stage 1–3A command, plus the updated smoke harness that chains table/JSON views.
- Default table renders now align with `docs/UI design.md`; JSON output mirrors the raw service payloads.
- `go test ./...` (with local caches) passes; new unit tests cover output fallbacks.

## Known Gaps / Open Work
1. **Testing Automation**
   - Expand the smoke harness and port it into CI now that it pulls IDs from raw JSON output.
2. **Stage 3B Planning**
   - Finalise the mutating workflow scope (job control, restores, license updates) and define guardrails before implementation begins.
3. **Extended Integration Coverage**
   - Backfill long-running soak tests for proxies, licensing, and traffic rule surfaces to verify behaviour under sustained workloads.

## Next Milestones
- Define Stage 3B (mutating workflows) scope and implementation plan.
- Backfill integration tests for rescan workflows (mocked sessions) to support CI gating.
- Wire release automation to publish the cross-platform binaries with checksums.
