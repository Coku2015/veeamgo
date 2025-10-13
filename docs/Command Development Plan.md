# VeeamGo Command Development Plan

## 1. Guiding Principles
- Ship functionality in thin, testable slices that deliver immediate operator value.
- Keep read operations ahead of write operations; every mutating command requires its read counterpart.
- Reuse kubectl-style verbs (`get`, `describe`, `create`, `delete`) for consistency and muscle memory.
- Treat the OpenAPI spec as the single source of truth; generate/update clients as part of the workflow.
- Centralise command keyword strings in code so naming adjustments stay low-risk and auditable.

## 2. Release Stages

### Stage 1 — Foundations (MVP)
**Status:** ✅ Completed (2025-10-11)
**Goals**: establish connectivity, configuration hygiene, and session awareness.  
**Endpoints**: `/api/oauth2/token`, `/api/v1/serverInfo`, `/api/v1/serverTime`, `/api/v1/sessions`.

| Command | Description | Notes |
| --- | --- | --- |
| `veeamgo login` | Authenticate and persist tokens securely | Support env vars, config file, and interactive prompt |
| `veeamgo logout` | Revoke cached tokens | Include `--all` to clear every profile |
| `veeamgo session get` | Show current session metadata | Display token expiry, scopes, server |
| `veeamgo server get info` | Return server build/edition data | Add `--json` flag |
| `veeamgo server get time` | Fetch server clock and timezone | Useful for drift detection |

**Exit Criteria**
- Config bootstrap wizard (`veeamgo login`) works on macOS/Linux/Windows.
- Session cache encrypted at rest or stored in OS keychain.
- Automated smoke test covers login, server info/time.

### Stage 2 — Infrastructure Management
**Status:** ✅ Completed (2025-10-11)
**Goals**: expose backup infrastructure inventory and maintenance actions early so operators can baseline environments.  
**Endpoints**: `/api/v1/backupInfrastructure/managedServers`, `/api/v1/backupInfrastructure/managedServers/{id}`, `/api/v1/backupInfrastructure/managedServers/{id}/rescan`, `/api/v1/backupInfrastructure/managedServers/rescan`, `/api/v1/backupInfrastructure/repositories`, `/api/v1/backupInfrastructure/repositories/{id}`, `/api/v1/backupInfrastructure/repositories/rescan`, `/api/v1/inventory/vmware/*`.

| Command | Description | Notes |
| --- | --- | --- |
| `veeamgo server get managed` | List managed servers | `--type` filter |
| `veeamgo server describe managed <id>` | Detailed server info | Show connection health |
| `veeamgo server rescan <id|--all>` | Trigger rescan | Provide wait/async options |
| `veeamgo repository get` | Inventory repositories | Display capacity metrics |
| `veeamgo repository describe <name>` | Detailed repo configuration | Highlight encryption |
| `veeamgo repository rescan <id|--all>` | Trigger repository rescan | Link to task sessions |
| `veeamgo inventory servers` | Browse managed servers/hosts | Uses non-deprecated POST inventory API with optional filters |

**Exit Criteria**
- Infrastructure scans show progress updates and final status.
- Warnings for unsupported operations (e.g., missing licenses) appear with remediation suggestions.
- Documentation updated with infrastructure operator examples.

### Stage 3 — Read-Only Visibility
**Status:** ⏳ In progress — Stage 3A scoping  
**Goals**: deliver Stage 3A read-only coverage across jobs, sessions, protected data, and active mounts before introducing any mutating flows. See `docs/Stage 3A Read-only Coverage.md` for the full endpoint inventory.  
**Endpoints** (P0): `/api/v1/jobs`, `/api/v1/jobs/{id}`, `/api/v1/jobs/states`, `/api/v1/sessions*`, `/api/v1/taskSessions*`, `/api/v1/tasks*`, `/api/v1/backups*`, `/api/v1/backupObjects*`, `/api/v1/restorePoints*`, `/api/v1/replicas*`, `/api/v1/replicaPoints*`, `/api/v1/restore/instantRecovery/*`, `/api/v1/dataIntegration*`, `/api/v1/backupBrowser/flr*`, `/api/v1/inventory*`.

| Command | Description | Notes |
| --- | --- | --- |
| `veeamgo job get` | List jobs (`--name`, `--type`, pagination) | Status badges and next-run info |
| `veeamgo job describe <name>` | Detailed job definition and last runs | Join with latest session summary |
| `veeamgo session list` | Enumerate job/scheduler sessions | Support `--type`, `--since`, `--limit` |
| `veeamgo session describe <id>` | Single session timeline | Include task counts and duration |
| `veeamgo session logs <id>` | Fetch session log lines | Provide tail/follow switches |
| `veeamgo task get [--session <id>]` | List task sessions | Reuse output columns from jobs |
| `veeamgo task describe <id>` | Task runtime and errors | Surface linked object/job |
| `veeamgo task logs <id>` | Task log entries | Filter by status for focused triage |
| `veeamgo inventory servers|objects|physical` | Browse inventory via POST APIs | Accept hierarchy and raw JSON filters |
| `veeamgo backup get [--job <id>]` | Inventory backups | Add retention, capacity columns |
| `veeamgo backup describe <name>` | Backup metadata | Merge repository and job refs |
| `veeamgo backup files <name>` | Enumerate backup files | Allow CSV/JSON export |
| `veeamgo backup objects <name>` | Protected objects per backup | Provide name filters |
| `veeamgo restore-point get [--backup <id>]` | List restore points | Show type and creation time |
| `veeamgo restore-point describe <name|id>` | Detailed restore point | Optionally expand disk inventory |
| `veeamgo restore-point disks <id>` | Restore point disks | Surface capacity and state |
| `veeamgo replica get` | Inventory replicas | Highlight source job and site |
| `veeamgo replica-point get [--replica <id>]` | Replica restore points | Indicate checkpoints and powershell IDs |
| `veeamgo restore mount get` | List active mounts (IR, FLR, data integration) | Group by platform and purpose |
| `veeamgo restore mount describe <id>` | Mount/session detail | Include switchover hints |
| `veeamgo restore mount sessions <mountId>` | Enumerate mount sessions | Link to originating job/session |

**Exit Criteria**
- All commands above implemented with help text, examples, and `--json` output parity.
- `internal/client` packages provide covered GET operations with unit tests and mock fixtures.
- Output tables follow `docs/UI design.md` typography; snapshot tests guard against regressions.
- Manual runbook (using `.env.test`) captures transcripts for jobs, sessions, backups, and mounts.
- Stage 3A document stays current as endpoints/commands evolve; Stage 3B (mutating flows) re-evaluated after sign-off.

### Stage 4 — Control Plane Operations
**Status:** 🔜 Planned  
**Goals**: introduce POST/PUT/DELETE flows for job lifecycle control and remediation tasks once Stage 3A read-only coverage ships.  
**Endpoints**: `/api/v1/jobs/{id}/start`, `/api/v1/jobs/{id}/stop`, `/api/v1/jobs/{id}/enable`, `/api/v1/jobs/{id}/disable`, `/api/v1/jobs/{id}/retry`, `/api/v1/jobs/{id}/clone`, `/api/v1/taskSessions/{id}/retry`, `/api/v1/backupInfrastructure/*/rescan`, `/api/v1/restore/instantRecovery/*/unmount`.

| Command | Description | Notes |
| --- | --- | --- |
| `veeamgo job start <id>` | Trigger an on-demand run | Require confirmation prompt or `--yes` |
| `veeamgo job stop <id>` | Stop a running job | Warn about partial results |
| `veeamgo job enable <id>` | Enable disabled job | Surface schedule preview |
| `veeamgo job disable <id>` | Disable job | Offer `--until <time>` for temporary disable |
| `veeamgo job retry <id>` | Retry job session | Expose retry strategy |
| `veeamgo job clone <id>` | Clone job definition | Allow overrides via flags/config |
| `veeamgo task retry <id>` | Retry failed task session | Link to parent job/session metadata |
| `veeamgo repository rescan <id|--all>` | Rescan repositories | Extend Stage 2 logic with task tracking |
| `veeamgo server rescan <id|--all>` | Rescan managed servers | Provide async wait flags |
| `veeamgo restore mount unmount <id>` | Tear down IR/FLR mount | Ensure graceful cleanup with `--force` |

**Exit Criteria**
- Mutating commands surface task/session IDs and track completion progress.
- Safety checks guard against accidental destructive actions; dry-run previews where API allows.
- Integration coverage exercises start/stop/retry and rescan flows against lab jobs.
- Error handling differentiates auth failures, validation issues, and transient infrastructure errors.

### Stage 5 — Restore Workloads
**Status:** 🔜 Planned
**Goals**: support instant recovery, full VM restore, and initial file-level recovery flows.  
**Endpoints**: `/api/v1/restore/instantRecovery/vSphere/vm`, `/api/v1/restore/instantRecovery/vSphere/vm/{id}/unmount`, `/api/v1/restore/vmRestore/vSphere`, `/api/v1/restore/flr`, `/api/v1/restore/flr/{sessionId}/validateCredentials`, `/api/v1/restore/flr/{sessionId}/unmount`.

| Command | Description | Notes |
| --- | --- | --- |
| `veeamgo restore instant-vm` | Launch instant recovery session | Require target compute/storage options |
| `veeamgo restore list-mounts` | Show active instant recovery mounts | Include cleanup hints |
| `veeamgo restore unmount <id>` | Unmount instant recovery session | Support `--force` |
| `veeamgo restore vm` | Full VM restore workflow | Provide dry-run summary |
| `veeamgo restore flr start` | Start file-level recovery session | Accept credentials/profile hints |
| `veeamgo restore flr validate <session>` | Validate guest credentials | Mask sensitive output |
| `veeamgo restore flr stop <session>` | Unmount FLR session | Graceful cleanup |

**Exit Criteria**
- Stateful operations emit progress updates and signal task IDs.
- Dry-run mode (where API permits) previews planned actions.
- Integration tests cover instant recovery start/unmount with mock lab.

### Stage 6 — Advanced Capabilities
**Status:** 🔜 Planned
**Goals**: finish coverage with licensing, reporting, automation niceties.  
**Endpoints**: `/api/v1/license`, `/api/v1/license/install`, `/api/v1/license/remove`, plus future extension endpoints.

| Command | Description | Notes |
| --- | --- | --- |
| `veeamgo license get` | Display license summary | Provide expiry alerts |
| `veeamgo license install --file` | Upload new license | Validate file path |
| `veeamgo license remove` | Remove current license | Enforce confirmation |
| `veeamgo report get sessions|tasks` | Export CSV/JSON snapshots | Accept time range |
| `veeamgo config profile` | Manage multiple server profiles | `create`, `list`, `use`, `delete` |

**Exit Criteria**
- Full CLI coverage matrix mapped to OpenAPI operations.
- CI pipelines enforce lint, unit, integration, and doc validation.
- Telemetry/analytics (if enabled) respect privacy opt-in.

## 3. Supporting Workstreams
- **OpenAPI Maintenance**: keep `docs/swagger/` split structure current; auto-generate client stubs when the spec changes.
- **Testing Infrastructure**: maintain a mock server, record/replay fixtures, and nightly integration runs against a staging VBR instance.
- **Documentation**: update `docs/` with walkthroughs per stage; include release notes and upgrade guides.
- **Release Automation**: GitHub Actions (or equivalent) building multi-platform binaries, signing artifacts, and publishing checksums.

## 4. Milestone Review Checklist
1. All commands for the stage implemented with help text and examples.
2. Unit tests ≥ 80% coverage for new packages; integration tests for API interactions.
3. User-facing docs updated, including changelog entry.
4. CLI adheres to logging, error, and output standards.
5. Security review completed for new surfaces (token handling, sensitive logging).

## 5. Dependencies & Assumptions
- Access to a non-production VBR environment for integration testing.
- Credentials managed through secure secrets storage during CI.
- Development team comfortable with Go, Cobra, REST API patterns, and Veeam domain concepts.

## 6. Open Questions
- What is the long-term story for plugin/extensibility (e.g., community commands)?
- Should we introduce a `veeamgo proxy` for reducing API chatter in high-latency environments?
- Do we need to support older API versions (v11) beyond best-effort compatibility?

Document owner: Engineering PM. Reviews required before each stage exits beta.
