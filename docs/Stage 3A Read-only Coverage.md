# Stage 3A — Read-Only Command Coverage

## Objective
Stage 3 is refocused on delivering the highest-value read-only functionality before we introduce any mutating flows. Using `docs/swagger/openapi.json` as the source of truth, this sub-stage implements `get` and `describe` commands across the endpoints that operators rely on for situational awareness of jobs, sessions, and data protection assets.

## Scope extracted from docs/swagger/openapi.json
### Primary coverage (P0)
| Status | Domain | API paths (GET) | CLI pattern seeds | Notes |
| --- | --- | --- | --- | --- |
| ✅ Completed | Job Inventory & States | `/api/v1/jobs`<br>`/api/v1/jobs/{id}` | `veeamgo job get`<br>`veeamgo job describe <name>` | Implement pagination (`skip`, `limit`), filtering (`--type`, `--name`), and surface status badges aligned with `pkg/output` helpers. |
| ✅ Completed | Run Sessions & Logs | `/api/v1/sessions`<br>`/api/v1/sessions/{id}`<br>`/api/v1/sessions/{id}/logs`<br>`/api/v1/sessions/{id}/taskSessions`<br>`/api/v1/taskSessions`<br>`/api/v1/taskSessions/{id}`<br>`/api/v1/taskSessions/{id}/logs`<br>`/api/v1/tasks`<br>`/api/v1/tasks/{id}` | `veeamgo session list`<br>`veeamgo session describe <id>`<br>`veeamgo session logs <id>`<br>`veeamgo task get [--session <id>]`<br>`veeamgo task describe <id>`<br>`veeamgo task logs <id>` | Unify log retrieval with streaming/truncation options, link task sessions back to their parent job/session, and reuse the redesigned table layout. |
| ✅ Completed | Backup Catalog & Objects | `/api/v1/backups`<br>`/api/v1/backups/{id}`<br>`/api/v1/backups/{id}/objects`<br>`/api/v1/backups/{id}/backupFiles`<br>`/api/v1/backups/{id}/backupFiles/{backupFileId}`<br>`/api/v1/backupObjects`<br>`/api/v1/backupObjects/{id}`<br>`/api/v1/backupObjects/{id}/restorePoints` | `veeamgo backup get [--job <id>]`<br>`veeamgo backup describe <name>`<br>`veeamgo backup files <name>`<br>`veeamgo backup objects <name>` | Provide size/retention summaries and allow export to JSON/CSV for large datasets; support cross-linking to jobs and repositories. |
| ✅ Completed | Restore Points & Replica Chains | `/api/v1/restorePoints`<br>`/api/v1/restorePoints/{id}`<br>`/api/v1/restorePoints/{id}/disks`<br>`/api/v1/replicas`<br>`/api/v1/replicas/{id}`<br>`/api/v1/replicas/{id}/replicaPoints`<br>`/api/v1/replicas/{replicaId}/replicaPoints/{id}`<br>`/api/v1/replicaPoints`<br>`/api/v1/replicaPoints/{id}` | `veeamgo restore-point get [--backup <id>]`<br>`veeamgo restore-point describe <name|id>`<br>`veeamgo restore-point disks <id>`<br>`veeamgo replica get`<br>`veeamgo replica describe <id>`<br>`veeamgo replica-point get [--replica <id>]` | Highlight retention chains, provide disk details on demand, and flag stale replicas. |
| ✅ Completed | Instant Recovery & Mount Monitoring | `/api/v1/restore/instantRecovery/vSphere/vm`<br>`/api/v1/restore/instantRecovery/vSphere/vm/{mountId}`<br>`/api/v1/restore/instantRecovery/hyperV/vm`<br>`/api/v1/restore/instantRecovery/hyperV/vm/{mountId}`<br>`/api/v1/restore/instantRecovery/azure/vm`<br>`/api/v1/restore/instantRecovery/azure/vm/{mountId}`<br>`/api/v1/restore/instantRecovery/azure/vm/{mountId}/sessions`<br>`/api/v1/restore/instantRecovery/azure/vm/{mountId}/switchoverSettings`<br>`/api/v1/restore/instantRecovery/vSphere/fcd`<br>`/api/v1/restore/instantRecovery/vSphere/fcd/{mountId}`<br>`/api/v1/dataIntegration`<br>`/api/v1/dataIntegration/{mountId}`<br>`/api/v1/backupBrowser/flr*` | `veeamgo restore mount get`<br>`veeamgo restore mount describe <id>`<br>`veeamgo restore mount sessions <mountId>`<br>`veeamgo restore browser entra|flr ...` | Normalise mount visibility across hypervisor types, expose switchover metadata, and align outputs with the UI mockups in `docs/UI design.md`. |

### Secondary coverage (P1)
| Status | Domain | API paths (GET) | CLI pattern seeds | Notes |
| --- | --- | --- | --- | --- |
| ✅ Completed | Credentials & Encryption | `/api/v1/credentials`<br>`/api/v1/credentials/{id}`<br>`/api/v1/cloudCredentials`<br>`/api/v1/cloudCredentials/{id}`<br>`/api/v1/cloudCredentials/{id}/helperAppliances`<br>`/api/v1/encryptionPasswords`<br>`/api/v1/encryptionPasswords/{id}`<br>`/api/v1/kmsServers`<br>`/api/v1/kmsServers/{id}` | `veeamgo credential get`<br>`veeamgo credential describe <id>`<br>`veeamgo credential helper-appliance list <id>`<br>`veeamgo encryption password get|describe` | Prioritise redaction of sensitive fields and tie into the existing credential profile UX. |
| ✅ Completed | Agent Management & Protection Groups | `/api/v1/agents/protectionGroups`<br>`/api/v1/agents/protectionGroups/{id}`<br>`/api/v1/agents/protectionGroups/{id}/discoveredEntities`<br>`/api/v1/agents/protectionGroups/{id}/discoveredEntities/{entityId}`<br>`/api/v1/agents/protectedComputers`<br>`/api/v1/agents/protectedComputers/{id}`<br>`/api/v1/agents/recoveryTokens*`<br>`/api/v1/agents/packages/*` | `veeamgo protection-group get|describe`<br>`veeamgo agent get`<br>`veeamgo agent describe <id>`<br>`veeamgo agent package list linux|unix` | Requires hierarchical output (group → entities) and download hints for agent packages. |
| ✅ Completed | Security & Audit | `/api/v1/securityAnalyzer/*`<br>`/api/v1/authorization/events`<br>`/api/v1/authorization/events/{id}`<br>`/api/v1/malwareDetection/events*`<br>`/api/v1/malwareDetection/yaraRules` | `veeamgo security analyzer get`<br>`veeamgo security events get|describe`<br>`veeamgo malware event get|describe` | Provide severity badges, default time-range filters, and pagination safeguards. |
| ✅ Completed | Configuration & Licensing | `/api/v1/license`<br>`/api/v1/license/sockets`<br>`/api/v1/license/instances`<br>`/api/v1/license/capacity`<br>`/api/v1/generalOptions`<br>`/api/v1/configBackup`<br>`/api/v1/trafficRules`<br>`/api/v1/globalExclusions/vm*`<br>`/api/v1/services` | `veeamgo license get`<br>`veeamgo options get`<br>`veeamgo exclusion vm get|describe`<br>`veeamgo service get` | Some Stage 1/2 commands cover portions; expand coverage with consistent formatting and diff-friendly output. |
| ✅ Completed | Inventory Browser Extensions | `/api/v1/inventory`<br>`/api/v1/inventory/{hostname}`<br>`/api/v1/inventory/unstructuredDataServers*`<br>`/api/v1/inventory/entraId/tenants*`<br>`/api/v1/inventory/activeDirectory/domains/{id}`<br>`/api/v1/adDomains*` | `veeamgo inventory servers`<br>`veeamgo inventory objects <hostname>`<br>`veeamgo inventory physical`<br>`veeamgo inventory physical-items <groupId>` | Harmonise filters across inventory types and ensure recursive fetches remain performant. Avoid deprecated VMware inventory endpoints; identify stable replacements before implementing. |
| ✅ Completed | Proxy & Transport Services | `/api/v1/backupInfrastructure/proxies`<br>`/api/v1/backupInfrastructure/proxies/{id}`<br>`/api/v1/backupInfrastructure/proxies/states`<br>`/api/v1/backupInfrastructure/managedServers/{id}/volumes`<br>`/api/v1/backupInfrastructure/managedServers/optionalComponents/defaults` | `veeamgo proxy get`<br>`veeamgo proxy describe <id>`<br>`veeamgo proxy states`<br>`veeamgo server volume list <id>`<br>`veeamgo server optional-components defaults` | Surface transport proxy inventory, capacity and OS add-ons so operators can validate workload routing. |
| ✅ Completed | Repository & Mount Infrastructure | `/api/v1/backupInfrastructure/scaleOutRepositories*`<br>`/api/v1/backupInfrastructure/repositories/states`<br>`/api/v1/backupInfrastructure/mountServers`<br>`/api/v1/backupInfrastructure/mountServers/{id}`<br>`/api/v1/backupInfrastructure/mountServers/default`<br>`/api/v1/backupInfrastructure/wanAccelerators*` | `veeamgo repository scaleout get|describe`<br>`veeamgo repository states`<br>`veeamgo mount-server get|describe|default`<br>`veeamgo wan-accelerator get|describe` | Complements Stage 2 by expanding to auxiliary infrastructure and ensuring repository health/state data is visible from the CLI. |

### Coverage checklist
- **P0 (Completed Stage 3A implementation)** — Jobs, Sessions/Tasks, Backups & Objects, Restore Points & Replicas, Instant Recovery & Mounts, Inventory Browser (servers/objects/physical), Data Integration, Backup Browser (FLR/Entra).
- **P1 (Completed)** — Credentials & Encryption, Agent Protection Groups, Security Analyzer & Audit Events, Configuration & Licensing, Additional Inventory Browser surfaces (Entra ID, AD, Unstructured Data), Proxy & Transport Services, Repository & Mount Infrastructure, Active Directory Domains.
- **Deferred / Stage 3B+** — Any POST/PUT/DELETE flows, download helpers (`/api/v1/deployment/*`, `/api/v1/exportlogs/*`), niche monitoring surfaces (e.g., `/api/v1/services`, `/api/v1/serverCertificate`) that are already indirectly covered in Stage 1 documentation or can wait for operator demand.

## Deliverables
- Cobra subcommands and flag wiring for every P0 endpoint group, including help text and usage examples that reflect the OpenAPI filters.
- `internal/client` additions (jobs, sessions, backups, restore points, replicas, mounts) with unit tests that mock HTTP GET flows.
- Output templates in `pkg/output` keeping the column order and typography from `docs/UI design.md`.
- Updated operator documentation (`docs/Progress Report.md`, command reference) outlining the new read-only commands and sample workflows.
- When the OpenAPI Summary reads “Get …” but the verb differs, implement the command under the read-only verb family (`get`/`describe`) while respecting the underlying HTTP method in the client.
- Maintain a backlog of POST-powered read operations whose Summary begins with “Get …”; these must surface as read-only CLI verbs even though the transport verb is POST:
  - `/api/oauth2/token` — Get Access Token
  - `/api/v1/inventory` — Get All Servers
  - `/api/v1/inventory/{hostname}` — Get Inventory Objects
  - `/api/v1/inventory/physical` — Get All Protection Groups
  - `/api/v1/inventory/physical/{protectionGroupId}` — Get Inventory Objects for Specific Protection Group
  - `/api/v1/backupInfrastructure/managedServers/cloudDirectorHosts` — Get vCenter Servers Attached to Cloud Director Server
  - `/api/v1/backupInfrastructure/managedServers/hyperVHosts` — Get Hyper-V Servers Managed by Hyper-V Cluster or SCVMM Server
  - `/api/v1/cloudCredentials/appRegistration` — Get Microsoft Entra ID Verification Code
  - `/api/v1/cloudCredentials/authenticate` — Get Google Authentication Information
  - `/api/v1/cloudBrowser` — Get Cloud Hierarchy
  - `/api/v1/restore/entraId/tenant/deviceCode` — Get User Code for Delegated Restore of Microsoft Entra ID Items
  - `/api/v1/restore/entraId/tenant/deviceCode/state` — Get Credentials for Delegated Restore of Microsoft Entra ID Items
  - `/api/v1/backupBrowser/entraIdTenant/{backupId}/restorePoints` — Get Restore Points of Microsoft Entra ID Tenant
  - `/api/v1/backupBrowser/entraIdTenant/{backupId}/browse` — Get Microsoft Entra ID Items
  - `/api/v1/backupBrowser/entraIdTenant/{backupId}/browse/{itemId}` — Get Microsoft Entra ID Item
  - `/api/v1/backupBrowser/entraIdTenant/{backupId}/browse/{itemId}/restorePoints` — Get Restore Points of Microsoft Entra ID Item

### Stage 3B (In Progress)
- UI-focused refinements tracked in `docs/Stage 3B Plan.md`, covering repository table readability, licensing summaries, job-type filters, verb-first aliases (`veeamgo get|describe <resource>`), and the job-scoped `restorepoint`/`replica` commands ahead of the Stage 3C inventory redesign.

## Acceptance Criteria
- All P0 commands support JSON output, pagination controls, and error handling consistent with Stage 1/2 conventions.
- Automated tests cover happy/error paths for new clients; snapshot/table verification prevents regressions in formatting.
- Manual runbook in `.env.test` environment exercises each P0 command with recorded CLI transcripts.
- OpenAPI split (`docs/swagger/paths/*`) stays in sync with the aggregated `docs/swagger/openapi.json`.

## Out of Scope
- Any POST/PUT/DELETE operations (start/stop, clone, rescan) remain deferred to a future Stage 3B.
- Non-GET endpoints for credential rotation, restore launches, or policy changes.
- UI redesign changes beyond those already approved in `docs/UI design.md`.
