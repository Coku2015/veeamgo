# VeeamGo Testing and Debugging Guide

## 1. Purpose
Provide a repeatable playbook for validating each development stage of VeeamGo, diagnosing issues quickly, and keeping quality gates enforceable in CI/CD.

## 2. Testing Strategy by Stage

### Stage 1 — Connectivity & Session
- **Focus**: OAuth2 flows, config bootstrap, server info retrieval.
- **Manual Checklist**
  ```bash
  veeamgo login --server https://lab-vbr:9419 --username admin --password ****** --insecure
  veeamgo session get
  veeamgo server get info --json
  veeamgo server get time
  veeamgo logout
  ```
- **Config-based shortcut**: Copy `docs/examples/veeamgo.config.example.yaml` to `~/.veeamgo/config`, update credentials (passwords accepted as `ENC:<base64>` values), and rerun `veeamgo server get info` to confirm automatic login/refresh works without an explicit `login` command.
- **Expected Outcomes**
  - `session get` displays access/refresh expiry and server identifier.
  - Server info/time commands work only when logged in; they fail gracefully otherwise.
  - Logout wipes cached tokens and removes entries from `~/.veeamgo/sessions.json`.
- **Debug Tips**
  - Run with `--debug` to inspect HTTP requests/responses (tokens redacted).
  - Inspect `~/.veeamgo/sessions.json` (encrypted JSON) for cached tokens; delete the file to force re-authentication.

### Stage 2 — Infrastructure
- **Focus**: managed servers, repositories, inventory rescans.
- **Manual Checklist**
  ```bash
  veeamgo server get managed
  veeamgo server describe managed <server-id>
  veeamgo server rescan --all --async
  veeamgo repository get --output json
  veeamgo repository rescan <repo-id> --wait
  veeamgo inventory servers --limit 5
  ```
- **Expected Outcomes**
  - Rescan commands log task progress and exit status.
  - Inventory queries fetch hierarchical paths and annotate health issues.

### Stage 3 — Jobs & Tasks
- **Focus**: job listing, describing, state aggregation, and lifecycle mutations.
- **Manual Checklist**
```bash
veeamgo job get --output table
veeamgo job get --output json --name smoke
veeamgo job describe "<job-name>"
  veeamgo job start <job-id> --yes
  veeamgo session list --limit 5
  veeamgo session describe <session-id>
  veeamgo session logs <session-id> --status Warning
  veeamgo task get --job <job-id>
  veeamgo task describe <task-id>
  veeamgo task logs <task-id>
  veeamgo job disable <job-id> --yes && veeamgo job enable <job-id> --yes
  ```
- **Expected Outcomes**
  - Lists support pagination and filtering flags.
  - Mutating commands emit task session IDs and block until task resolves (unless `--async`).
  - Exit codes signal success (0), partial failure (3), not found (4), validation (5).
- **Debug Tips**
  - Use `--trace` to print request/response bodies for problematic endpoints.
  - Capture task logs via `veeamgo task logs <id> --status Error > task.log`.

### Stage 4 — Backups & Restore Points
- **Focus**: backup inventory, file/object enumeration, restore point metadata.
- **Manual Checklist**
  ```bash
  veeamgo backup get --job <job-id>
  veeamgo backup describe <backup-id>
  veeamgo backup get files <backup-id> --output json > files.json
  veeamgo restore-point get --backup <backup-id>
  veeamgo restore-point describe <restore-point-id>
  veeamgo backup files "<backup-name>"
  veeamgo backup objects "<backup-name>"
  veeamgo restore-point describe "<restore-point-name>" --backup <backup-id>
  veeamgo restore-point disks <restore-point-id>
  ```
- **Expected Outcomes**
  - Large datasets use streaming pagination; CLI prints progress indicator if >500 rows.
  - JSON output validates against embedded schema tests.
  - Missing resources return exit code 4 with “Not Found” guidance.

### Stage 5 — Restore Workflows
- **Focus**: instant recovery, full VM restore, FLR sessions.
- **Manual Checklist**
  ```bash
  veeamgo restore instant-vm --restore-point <rp-id> --resource-pool QA --datastore NVMe --folder QA --network "VM Network" --yes
  veeamgo restore list-mounts
  veeamgo restore unmount <mount-id> --yes
  veeamgo restore vm --restore-point <rp-id> --host esx-01 --datastore NVMe --path QA/Restores --dry-run
  veeamgo restore flr start --backup <backup-id> --restore-point <rp-id> --session-name smoke
  veeamgo restore flr validate <session-id> --credentials domain\\user
  veeamgo restore flr stop <session-id>
  ```
- **Expected Outcomes**
  - Dry-run outputs show placement plan without mutating environment.
  - Active mounts list includes creation time, owner, protection for accidental deletion.
  - FLR sessions enforce credential masking and timeouts.

### Stage 6 — Advanced & Reporting
- **Focus**: license operations, reporting exports, profile management.
- **Manual Checklist**
  ```bash
  veeamgo license get
  veeamgo license install --file ./license.lic --yes
  veeamgo license remove --yes
  veeamgo report get sessions --from 2024-01-01 --to 2024-01-31 --output csv
  veeamgo config profile list
  ```
- **Expected Outcomes**
  - License commands warn before downtime-impacting changes.
  - Reports respect time range, timezone, format flags, and exit 0 only when artifact written.

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
- **Auth Fails**: check clock skew (`veeamgo server get time`), verify TLS trust, ensure token cache not stale (`veeamgo session clear --all`).
- **API 429/503**: CLI should back off; if persistent, enable `--debug` to inspect `Retry-After`, adjust concurrency.
- **Slow Reads**: use `--limit` to constrain result size, check network latency metrics printed in debug mode.
- **Restore Errors**: confirm resource pool/datastore names, validate user permissions, review task logs via `veeamgo task describe`.

## 6. Sample Automation Script
```bash
#!/usr/bin/env bash
set -euo pipefail

echo "== Stage 1 Regression =="
veeamgo login --server "${VEEAMGO_SERVER}" --username "${VEEAMGO_USERNAME}" --password "${VEEAMGO_PASSWORD}" --insecure
veeamgo server get info --output json | jq '.build'
veeamgo job get --limit 5 --output table
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
