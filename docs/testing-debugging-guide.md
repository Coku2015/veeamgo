# VeeamGo Testing and Debugging Guide

## 1. Purpose
Provide a repeatable playbook for validating each development stage of VeeamGo, diagnosing issues quickly, and keeping quality gates enforceable in CI/CD.

## 2. Testing Strategy by Stage

### Stage 1 — Connectivity & Session
- **Focus**: OAuth2 flows, config bootstrap, server info retrieval.
- **Manual Checklist**
  ```bash
  veeamgo login --server https://lab-vbr:9419 --username admin --password ****** --insecure
  veeamgo get session
  veeamgo get server info --output json
  veeamgo get server time
  veeamgo logout
  ```
- **Config-based shortcut**: Copy `docs/examples/veeamgo.config.example.yaml` to `~/.veeamgo/config`, update credentials (passwords accepted as `ENC:<base64>` values), and rerun `veeamgo get server info` to confirm automatic login/refresh works without an explicit `login` command.
- **Expected Outcomes**
  - `get session` displays access/refresh expiry and server identifier.
  - Server info/time commands work only when logged in; they fail gracefully otherwise.
  - Logout wipes cached tokens and removes entries from `~/.veeamgo/sessions.json`.
- **Debug Tips**
  - Run with `--debug` to inspect HTTP requests/responses (tokens redacted).
  - If start-up feels slower against a stable server version, capture the negotiated revision from the profile (`api_version`) and pass `--api-version <revision>` during login to skip the handshake.
  - Inspect `~/.veeamgo/sessions.json` (encrypted JSON) for cached tokens; delete the file to force re-authentication.

### Stage 2 — Infrastructure
- **Focus**: managed servers, repositories, inventory rescans.
- **Manual Checklist**
```bash
veeamgo get managedserver
veeamgo describe managedserver --id <server-id>
veeamgo rescan server --all --wait
veeamgo get repository --output json
veeamgo rescan repository --id <repo-id> --wait
veeamgo get sobr --limit 5
veeamgo describe sobr --name "<sobr-name>"
veeamgo get wanaccelerator --limit 5
veeamgo describe wanaccelerator --name "<wan-name>"
veeamgo get generaloption
veeamgo get inventory virtualinfra --limit 5
  # Lab-only smoke: requires an available target host/credential
  veeamgo add managedserver vsphere --name vc01.lab.local --username svc-vbr --password ****** --wait --yes
  # Lab-only smoke (deployment kit): certificate-based onboarding skips credentials
  # veeamgo add managedserver windows --name winrepo-cert.lab.local --connect-mode Certificate --wait --yes
  # Lab-only smoke (Linux): all parameters supplied explicitly
  # veeamgo add managedserver linux --name repo01.lab.local --description "Repo" --credentials-storage Permanent --ssh-fingerprint "ssh-rsa 3072 AAAA..." --wait --yes
  ```
- **Expected Outcomes**
  - Rescan commands log task progress and exit status.
  - `veeamgo get inventory ...` queries fetch hierarchical paths and annotate health issues across virtual, unstructured, and agent protection groups.
  - Scale-out repository commands highlight placement policy, capacity/archive tier settings, and extent health at a glance.
  - WAN accelerator commands surface cache sizing, traffic ports, and bandwidth mode for each accelerator.
  - `get generaloption` prints yes/no toggles for notifications and SIEM integrations in a describe-style table.
  - Managed server provisioning prints the session ID, auto-generates descriptions when omitted, and reports the server ID when run with `--wait`. Certificate-based onboarding (`--connect-mode Certificate`) assumes the deployment kit is already installed and rejects credential flags. Linux SingleUse mode requires fingerprint plus the `--single-use-*` credentials and succeeds without storing a permanent credential.

### Stage 3 — Jobs & Tasks
- **Focus**: job listing, describing, state aggregation, and lifecycle mutations.
- **Manual Checklist**
```bash
veeamgo get job --output table
veeamgo get job --output json --name smoke
  veeamgo describe job --name "<job-name>"
  veeamgo start job --id "<job-id>" --yes
  veeamgo get session --limit 5
  veeamgo describe session --id "<Session ID>"
  veeamgo get session logs --id "<Session ID>" --status Warning
  veeamgo get task --session-id "<Session ID>"
  veeamgo describe task --id "<Task ID>"
  veeamgo get task logs --id "<Task ID>"
  veeamgo get malwaredetectionevent --limit 5
  veeamgo get yararule
  veeamgo disable job --id "<job-id>" --yes && veeamgo enable job --id "<job-id>" --yes
  ```
- **Expected Outcomes**
  - Lists support pagination and filtering flags.
  - Mutating commands emit task session IDs and block until task resolves (unless `--async`).
  - Exit codes signal success (0), partial failure (3), not found (4), validation (5).
  - Malware detection events surface severity, engine, detection timestamps; YARA rules list uploaded rule files.
- **Debug Tips**
  - Use `--trace` to print request/response bodies for problematic endpoints.
  - Capture task logs via `veeamgo get task logs --id <id> --status Error > task.log`.

### Stage 4 — Backups & Restore Points
- **Focus**: backup inventory, file/object enumeration, restore point metadata.
- **Manual Checklist**
  ```bash
  veeamgo get backup list --job <job-id>
  veeamgo get backup list --output json > backups.json
  veeamgo get backup files --backup "<backup-name>" --output json > files.json
  veeamgo get backup objects --backup "<backup-name>"
  veeamgo get restorepoint --backup "<backup-name>"
  veeamgo describe restorepoint "<restore-point-name>" --backup "<backup-name>"
  ```
- **Expected Outcomes**
  - Large datasets use streaming pagination; CLI prints progress indicator if >500 rows.
  - JSON output validates against embedded schema tests.
  - Missing resources return exit code 4 with “Not Found” guidance.

### Stage 5 — Restore Workflows
- **Status**: Restore orchestration (instant recovery, VM restore, FLR) is not yet exposed via `veeamgo`. Continue to exercise these flows through the VBR UI or REST API until the CLI adds coverage (tracked in `docs/command-development-roadmap.md`).

### Stage 6 — Advanced & Reporting
- **Focus**: license utilisation insights.
- **Manual Checklist**
  ```bash
  veeamgo get license
  veeamgo get license sockets --limit 5
  veeamgo get license instances --limit 5
  veeamgo get license capacity
  ```
- **Expected Outcomes**
  - License commands emit human-readable tables and JSON suitable for dashboards.

## 3. Automated Testing
- **Unit Tests**
  - Use `go test ./...` with mocks for API clients.
  - Cover command factories, flag parsing, output transformers.
- **Integration Tests**
  - `go test -tags=integration ./integration/...` hitting lab VBR server (credentials via environment).
  - Provide sanitized record/replay fixtures for CI when lab unavailable.
- **Lint & Static Analysis**
  - `golangci-lint run ./...`
  - `spectral lint docs/swagger/openapi.json`
- **Smoke Pipeline**
  - GitHub Action (or similar) building multi-platform binaries, running unit + integration suites, publishing coverage.

## 4. Debugging Toolkit
- `--debug` flag enables request headers, status codes, latency metrics.
- `--trace` dumps request/response bodies to stderr (tokens redacted).
- `--log-file <path>` captures extended diagnostics for support bundles.
- `VEEAMGO_DEBUG=true` environment flag turns on structured JSON logs.
- `veeamgo diag collect` (future command) to package logs, config metadata, and environment info.

## 5. Troubleshooting Patterns
- **Auth Fails**: check clock skew (`veeamgo get server time`), verify TLS trust, and clear cached tokens with `veeamgo logout --all` when in doubt.
- **API 429/503**: CLI should back off; if persistent, enable `--debug` to inspect `Retry-After`, adjust concurrency.
- **Slow Reads**: use `--limit` to constrain result size, check network latency metrics printed in debug mode.
- **Restore Errors**: confirm resource pool/datastore names, validate user permissions, review task logs via `veeamgo describe task --id <Task ID>`.

## 6. Sample Automation Script
```bash
#!/usr/bin/env bash
set -euo pipefail

echo "== Stage 1 Regression =="
veeamgo login --server "${VEEAMGO_SERVER}" --username "${VEEAMGO_USERNAME}" --password "${VEEAMGO_PASSWORD}" --insecure
veeamgo get server info --output json | jq '.build'
veeamgo get job --limit 5 --output table
veeamgo logout

echo "All good ✅"
```

## 7. Reporting Bugs
- Include CLI version (`veeamgo version`), OS, VBR version, command executed, flags, and redacted logs.
- Attach API trace (`--trace --log-file bug.log`) when possible.
- Tag issues with component label (`auth`, `jobs`, `restore`, `infra`).

## 8. References
- Veeam REST API Reference: https://helpcenter.veeam.com/references/vbr/13/rest/
- CLAUDE.md for project-wide language and architecture guidance.
