# VeeamGo CLI Project Requirements

## 1. Overview
- **Project Name**: VeeamGo (CLI for Veeam Backup & Replication REST API v1.3)
- **Purpose**: Provide operations teams with a kubectl-inspired command line that automates day-to-day backup administration, restores, and infrastructure visibility.
- **Design Principles**: Simple verbs, automation friendly output, secure-by-default authentication, staged delivery aligned with real API capabilities.

## 2. Objectives
- Deliver fast read-focused workflows first (`veeamgo <resource> get ...`) before adding mutating commands.
- Mirror the official Veeam REST API v1.3 schema; never invent fields that are not exposed by the platform.
- Keep all source artifacts in English, including documentation, messages, and examples.
- Provide both human readable tables and structured JSON output for scripting.
- Integrate safely with production by enforcing HTTPS, token hygiene, and explicit opt-in for insecure scenarios.

## 3. Scope
1. **Authentication & Session Lifecyle**
   - Login with OAuth2 password grant (`veeamgo login`), logout, token refresh, and session inspection.
2. **Backup Job Insights**
   - Read operations: list/show/filter job inventory while surfacing status metadata from the job states endpoint.
   - Control plane (start/stop/enable/disable/retry) delivered once read APIs are stable.
3. **Backup Data & Restore Points**
   - Browse backups, related objects/files, and restore points with optional scoping by job or backup id.
4. **Restore Workflows**
   - Instant VM recovery, full restore, and file-level restore sessions with lifecycle commands (mount, status, teardown).
5. **Infrastructure Inventory**
   - Managed servers, repositories, proxies, inventory browsers, and rescans.
6. **Monitoring & Reporting**
   - Sessions, task sessions, license info, server metadata, and health indicators.
7. **Configuration & UX**
   - Support config files and env vars, default profile handling, pluggable output adapters (table/JSON), verbosity flags, and contextual help.
   - Auto-login/refresh: when credentials are present in the profile (or `VEEAMGO_PASSWORD` env), commands transparently acquire/refresh tokens without manual `login`/`logout` interaction.
   - Ship a documented sample config (`docs/examples/veeamgo.config.example.yaml`) to accelerate customer onboarding.

## 4. Functional Requirements
- Commands follow the shape `veeamgo <resource> <verb>` with `get` preferred for read-only flows.
- Sub-command keywords are centrally defined (see `cmd/names.go`) so branding-driven naming tweaks remain low risk.
- Each resource exposes:
  - `get`/`list` for collections (supports `--output table|json`, `--filter` patterns).
  - `describe` for detailed views (`veeamgo job describe <id>`).
  - Optionally scoped listing flags such as `--job <id>` or `--server <id>`.
- Early milestones must include:
  1. `login`, `logout`, `session get`, `session tokens`
  2. `job get`, `job describe`
  3. `session get`, `task get`, `task describe`
- Mutating verbs (`start`, `stop`, `create`, `delete`, etc.) require confirmation prompts or `--yes` bypass and must surface task session tracking.

## 5. Non-Functional Requirements
- **Performance**: Typical `get` commands should complete within 2 seconds under nominal API latency; CLI startup < 1 second; memory footprint < 50 MB.
- **Security**: Enforce HTTPS; validate TLS certs by default (allow `--insecure` for labs); store tokens securely (system keychain or encrypted file); redact secrets in logs.
- **Reliability**: Automatic token refresh, robust retry/backoff for transient network issues, clear error taxonomy.
- **Portability**: Deliver binaries for Windows 10+, macOS 10.15+, and mainstream Linux on amd64 and arm64.
- **Release Process**: Provide reproducible cross-platform build commands (macOS/Linux/Windows, amd64/arm64) and publish artifacts per release.
- **Observability**: Structured logging with adjustable levels (`error`, `warn`, `info`, `debug`, `trace`) and optional API request tracing.
- **API Hygiene**: Reject endpoints tagged `deprecated` and treat Summary semantics as the source of truth—if an operation states “Get …” it must surface as a read-only CLI command even when the HTTP verb is POST.

## 6. API Specification Management
- Split the monolithic `docs/swagger.json` into the following structure to improve maintainability:
  ```
  docs/swagger/
    openapi.json
    paths/
      jobs.json
      infra-servers.json
      infra-proxies.json
      ...
    components/
      schemas.json
      responses.json
  ```
- Ensure `openapi.json` references modular components using valid `$ref` pointers.
- Use automated checks (e.g., `spectral`, `openapi-cli`) to validate the split specification.
- Track API version compatibility (v1.3 primary, best-effort v1.2/v1.1 fallback) and highlight any endpoint deviations.

## 7. Dependencies & Tooling
- **Language**: Go 1.19+
- **CLI Framework**: `spf13/cobra`
- **HTTP Client**: Go `net/http` with custom middleware or `resty` for convenience.
- **Configuration**: `spf13/viper` for file/env handling.
- **Output Rendering**: table writer (e.g., `github.com/jedib0t/go-pretty/table`) plus JSON marshalling.

## 8. Deliverables
- Cross-platform binaries and container image.
- Source tree with documentation, split OpenAPI spec, unit/integration tests, and example scripts.
- User manual, troubleshooting guide, API coverage matrix, and changelog.

## 9. Risks & Mitigations
- **API Drift**: Monitor Veeam release notes; add automated compatibility tests; gate breaking changes behind feature flags.
- **Authentication Complexity**: Provide detailed diagnostics, built-in token cache viewer, and fallback to password prompts when env vars absent.
- **Restore Safety**: Guard destructive operations with environment tagging, dry-run previews, and cleanup reminders.
- **Testing Availability**: Maintain mock server and record/replay fixtures to reduce dependence on lab hardware.

## 10. Success Criteria
- Coverage of >90% daily operator read workflows by Stage 3.
- Positive user feedback on command discoverability and output clarity.
- Sustained uptime (24h) without crashes or token refresh failures during soak tests.
- Automated CI pipeline executing lint, unit, and integration suites with <10 minute turnaround.
