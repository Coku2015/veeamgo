# VeeamGo Command Development Roadmap

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
| `veeamgo get session` | Show current session metadata | Display token expiry, scopes, server |
| `veeamgo get server info` | Return server build/edition data | Add `--output json` flag |
| `veeamgo get server time` | Fetch server clock and timezone | Useful for drift detection |

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
| `veeamgo get managedserver` | List managed servers | `--type` filter |
| `veeamgo describe managedserver --id <id>` | Detailed server info | Show connection health |
| `veeamgo rescan server <id|--all>` | Trigger rescan | Provide wait/async options |
| `veeamgo get repository` | Inventory repositories | Display capacity metrics |
| `veeamgo describe repository --name <name>` | Detailed repo configuration | Highlight encryption |
| `veeamgo rescan repository --id <id>|--all` | Trigger repository rescan | Link to task sessions |
| `veeamgo get sobr` | Inventory scale-out repositories | Show placement, extent count, tier usage |
| `veeamgo describe sobr --name <name>` | Detailed SOBR configuration | Include extents, tier policies, advanced settings |
| `veeamgo get wanaccelerator` | Inventory WAN accelerators | Expose cache sizing and host details |
| `veeamgo describe wanaccelerator --name <name>` | Detailed WAN accelerator configuration | Show traffic port, streams, cache configuration |
| `veeamgo get generaloption` | Global notification & SIEM options | Describe-style table highlights toggles |
| `veeamgo get inventory virtualinfra` | Browse managed servers/hosts | Uses non-deprecated POST inventory API with optional filters |

**Exit Criteria**
- Infrastructure scans show progress updates and final status.
- Warnings for unsupported operations (e.g., missing licenses) appear with remediation suggestions.
- Documentation updated with infrastructure operator examples.

### Stage 3 — Read-Only Visibility
**Status:** ✅ Completed (2025-10-14)  
**Highlights**: Delivered the verb-first read-only experience (jobs, sessions, tasks, backups, replicas, malware, WAN/SOBR, global options) with refreshed help text, standardised tables, and JSON passthrough parity. Smoke coverage captures canonical help/success/error flows and `docs/ui-design-reference.md` now hosts the authoritative UI catalogue. Archived summaries live in `docs/archived-stage-3a-read-only.md` and `docs/archived-stage-3b-ui-adjustments.md`.

### Stage 4 — Resource Provisioning (Add/Edit/Remove)
**Status:** 🔜 Planned  
**Goals**: introduce create/update/delete flows for the core resource set (repositories, proxies, jobs, protection groups, credentials) while enforcing validation and safety prompts.
**Endpoints**: `/api/v1/backupInfrastructure/managedServers`, `/api/v1/backupInfrastructure/repositories`, `/api/v1/backupInfrastructure/proxies`, `/api/v1/jobs`, `/api/v1/agents/protectionGroups`, `/api/v1/credentials`, plus child resources for extent membership and SOBR layout.

| Command Seed | Description | Notes |
| --- | --- | --- |
| `veeamgo add repository` | Create repositories (local, SOBR extent, cloud) | Support spec via flags or manifest file |
| `veeamgo add managedserver <type>` | Provision managed servers (vSphere, Windows) | Auto-creates credentials and descriptions; offers `--wait` to track the session |
| `veeamgo edit repository --name <name>` | Update capacity/tiering settings | Validate impact and support dry-run diff |
| `veeamgo remove repository --name <name>` | Delete repository with guard rails | Require explicit confirmation |
| `veeamgo add proxy` / `remove proxy` | Manage backup proxies | Handle transport mode, task limits |
| `veeamgo add job` / `edit job` / `remove job` | CRUD for backup/replica jobs | Offer manifest-driven specification |
| `veeamgo add protection-group` / `edit protection-group` | Manage agent protection scope | Include discovered entity options |
| `veeamgo credential add|update|remove` | Manage stored credentials | Redact secrets, support keychain storage |

**Exit Criteria**
- CRUD commands emit clear previews (`--dry-run`) and require confirmation for destructive actions.
- Validation errors surface actionable API feedback with field-level context.
- Unit tests cover payload assembly; integration cases exercise repository/job creation against the lab environment.
- Documentation provides manifest examples and rollback guidance.

### Stage 5 — Operational Controls (Start/Stop/Restore)
**Status:** 🔜 Planned  
**Goals**: orchestrate lifecycle actions once provisioning exists—starting/stopping jobs, enabling/disabling schedules, retrying tasks, and managing restore operations (instant recovery, FLR, unmount).
**Endpoints**: `/api/v1/jobs/{id}/start`, `/api/v1/jobs/{id}/stop`, `/api/v1/jobs/{id}/enable`, `/api/v1/jobs/{id}/disable`, `/api/v1/jobs/{id}/retry`, `/api/v1/taskSessions/{id}/retry`, `/api/v1/restore/instantRecovery/*`, `/api/v1/restore/flr/*`, `/api/v1/restore/*/unmount`.

| Command Seed | Description | Notes |
| --- | --- | --- |
| `veeamgo start job --name <name>` | Trigger on-demand job run | Output task/session IDs; optional `--wait` |
| `veeamgo stop job --name <name>` | Stop an active job | Warn about partial backups |
| `veeamgo enable job --name <name>` / `disable job` | Toggle schedules | Support temporary disable windows |
| `veeamgo retry job --name <name>` | Retry latest failed session | Offer `--session-id` override |
| `veeamgo retry task --id <id>` | Retry individual task session | Link back to parent job |
| `veeamgo restore instant-vm start|unmount` | Manage instant recovery | **Planned** – provide target/cleanup flags |
| `veeamgo restore flr start|validate|stop` | Control file-level recovery sessions | **Planned** – mask secrets in output |

**Exit Criteria**
- Actions report progress (polling or async handoff) and recommend follow-up commands (`session describe`, etc.).
- Safety prompts or `--yes` flags prevent accidental disruption.
- Integration tests exercise start/stop/retry logic against lab jobs and validate mount lifecycle.

### Stage 6 — Advanced Workflows & Specialized Operations
**Status:** 🔜 Planned  
**Goals**: finish the command surface with specialised POST/PUT flows (security analyzer, malware scans, license installation, reporting, automation helpers) and polish the operator ergonomics.
**Endpoints**: `/api/v1/securityAnalyzer/*`, `/api/v1/malwareDetections/*`, `/api/v1/license/*`, `/api/v1/reporting/*`, `/api/v1/automation/*`, plus any remaining niche endpoints surfaced by product.

| Command Seed | Description | Notes |
| --- | --- | --- |
| `veeamgo security analyzer start|suppress` | Run analyzer, manage best-practice states | Track suppression expiry |
| `veeamgo malware scan submit` | Kick off on-demand malware scans | Accept restore point/job scope |
| `veeamgo license install|remove` | Manage license files | **Planned** – validate checksum, prompt confirmation |
| `veeamgo report export sessions|tasks` | Generate CSV/JSON reports | Support scheduling via flags |
| `veeamgo automation import|export` | Move configuration bundles | Support dry-run and diff output |

**Exit Criteria**
- Advanced commands integrate with existing logging/telemetry standards and respect redaction rules.
- Documentation includes runbooks for security, licensing, and automation admins.
- CI adds targeted regression coverage for each specialised flow.

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
