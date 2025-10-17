# VeeamGo User Guide

*Updated: 2025‑10‑13*

---

## 1. CLI Overview

`veeamgo` is a command line client for Veeam Backup & Replication (VBR). It exposes inventory and status information via a verb‑first grammar inspired by `kubectl`:

```
veeamgo <verb> <resource> [subcommand] [flags]
```

- **Primary verbs**: `get`, `describe`, `rescan`, `add`, `edit`, `enable`, `disable`, `clone`, `delete`, `start`, `stop`, `retry`
- **Global flags** (available everywhere):
  | Flag | Description |
  | --- | --- |
  | `--config <path>` | Override config file (default `~/.veeamgo/config.yaml`). |
  | `--profile <name>` | Select an existing profile; defaults to the active profile. |
  | `--output table|json` | Human‑readable tables (`table`, default) or raw API payloads (`json`). |

### Timezone Handling
All timestamps rendered in table/describe output use the VBR server’s timezone and are formatted as:

```
YYYY-MM-DD HH:MM:SS (+HH:MM)
```

Example: `2025-10-13 14:02:18 (+08:00)`

---

## 2. Authentication & Profiles

### Login
```
veeamgo login \
  --server vbr.example.com \
  [--port 9419] \
  --username administrator \
  [--password *****] \
  [--insecure] \
  [--scope <oauth-scopes>] \
  [--save-password] \
  [--set-default]
```

- Prompts for a password if `--password` is omitted.
- Stores a session token in `~/.veeamgo/sessions.json` (encrypted).
- `--save-password` persists the password in `~/.veeamgo/config.yaml` for silent refresh.
- `--port` overrides the REST API port when the server does not listen on the default `9419`.

### Logout
```
veeamgo logout          # Clears the active profile session
veeamgo logout --all    # Removes all cached sessions
```

---

## 3. Command Reference

### 3.1 `get` Commands (Table + JSON)

Tables always render header rows, even when the dataset is empty.

#### Server
```
veeamgo get server info
veeamgo get server time
veeamgo get managedserver [--type <Type>] [--name <pattern>] [--limit N]
```
- `info`: VBR server metadata.
- `time`: Server clock; exact timezone metadata is also shown.
- `managedserver`: Managed servers; filter by type/name, limit results.

##### Describe
```
veeamgo describe managedserver --id "<Managed Server ID>"
```
Detailed view for a managed server; shows type, status, description, and raw payload via `--output json`.

##### Add
```
veeamgo add managedserver vsphere \
  --name vc01.example.com \
  --username svc-vbr \
  [--password *****] \
  [--thumbprint <TLS-thumbprint>] \
  [--wait] \
  [--yes]

veeamgo add managedserver windows \
  --name winrepo01.example.com \
  [--username administrator] \
  [--password *****] \
  [--connect-mode Credential|Certificate] \
  [--wait] \
  [--yes]

veeamgo add managedserver linux \
  --name repo01.example.com \
  --ssh-fingerprint "ssh-rsa 3072 AAAAB3NzaC1yc2EAAAADAQABAAABAQ..." \
  [--username veeam] \
  [--password *****] \
  [--connect-mode Credential|SingleUse|Certificate] \
  [--single-use-username repo] \
  [--single-use-password *****] \
  [--single-use-private-key "$(cat key.pem)"] \
  [--wait] \
  [--yes]
```
- If `--description` is omitted, the CLI generates a friendly description with the creation date.
- Provide `--credentials-id` to reuse an existing record; otherwise the CLI creates standard credentials from `--username`/`--password`.
- Leave `--password` empty to be prompted securely.
- `--wait` blocks until the provisioning session finishes; otherwise the command prints the session ID for follow-up (`veeamgo describe session --id <ID>`).
- `--connect-mode` defaults to `Credential`, which uses a stored account (`--username`/`--password` or `--credentials-id`).
- Set `--connect-mode Certificate` when the host has a deployment kit installed; in that mode omit all credential flags and the CLI submits the certificate-based request automatically.
- For Linux, the CLI auto-retrieves the SSH fingerprint when omitted and asks you to confirm it; provide `--ssh-fingerprint` to skip the prompt. Use `--connect-mode SingleUse` to supply ephemeral SSH credentials with the `--single-use-*` flags (password/private-key). Certificate mode may prompt for a six-digit handshake code when re-adding factory appliances; once entered (or passed via `--handshake-code`), the CLI re-fetches and accepts the fingerprint automatically.

#### Repository
```
veeamgo get repository [--type <Type>] [--name <pattern>] [--limit N]
```
Lists repositories with capacity/free/used metrics.

##### Describe
```
veeamgo describe repository --name "<Repository Name>"
```
Detailed view including raw YAML configuration.

#### Proxy
```
veeamgo get proxy [--name <pattern>] [--type <Type>] [--host-id <ID>] [--sort <column>] [--desc] [--limit N]
```

##### Describe
```
veeamgo describe proxy --name "<Proxy Name>" [--type <Type>]
```

#### Jobs
```
veeamgo get job \
  [--name <pattern>] \
  [--type <EJobType> ...] \
  [--status <Status>] [--result <Result>] \
  [--workload <Platform>] [--repository <Repository Name>] \
  [--high-priority] \
  [--since <RFC3339>] [--before <RFC3339>] \
  [--after-job <Name>] \
  [--sort <column>] [--desc] \
  [--limit N]
```
Repeat `--type` to combine multiple `EJobType` values (for example `VSphereBackup`, `BackupCopy`). `--sort` columns include `name`, `lastRun`, etc.

```
veeamgo get job history \
  --name "<Job Name>" \
  [--type <SessionType> ...] \
  [--state <State> ...] \
  [--result <Result> ...] \
  [--created-since <RFC3339>] [--created-before <RFC3339>] \
  [--ended-since <RFC3339>] [--ended-before <RFC3339>] \
  [--sort <column>] [--desc] \
  [--limit N]
```
Outputs the same session summary table as `veeamgo get session`, scoped to the specified job.

##### Describe
```
veeamgo describe job "<Job Name>"
```
Shows schedules, repository bindings, last session, etc.

##### Add / Edit
```
veeamgo add job --from path/to/spec.yaml --yes
veeamgo add job --set name="My Job" --set type=VSphereBackup --dry-run
veeamgo add job --template VSphereBackup --template-variant minimal --dry-run

veeamgo edit job --name "My Job" --from path/to/updated.yaml --yes
veeamgo edit job --id <Job ID> --set description="Nightly policy" --dry-run
veeamgo edit job --name "My Job" --from-live
veeamgo edit job --name "My Job" --template VSphereBackup --set storage.backupRepositoryId=<UUID>
```
- `--from` accepts YAML or JSON blueprints that mirror the REST API payload. Combine with `--set key=value` overrides for quick tweaks.
- `--dry-run` validates the blueprint and prints the resolved payload without modifying the job.
- `--template` seeds the blueprint with a built-in starter (see `veeamgo template job --list` for options). Use `--template-variant` to choose `minimal` or `full` variants as needed.
- `--from-live` exports the current configuration to `.veeamgo/cache/jobs/<job-id>.yaml` (override with `--template-path`).
- `--yes` bypasses the confirmation prompt; otherwise the CLI asks before applying changes.

###### Template helpers
```
veeamgo template job --list
veeamgo template job --type VSphereBackup --write ./vmware.yaml
veeamgo template job --type VSphereBackup --no-comments
```
- Templates are YAML blueprints embedded in the CLI; start from them when crafting new jobs.
- `--write` saves the template to disk (defaults to stdout). `--no-comments` strips guidance lines for cleaner output.
- Combine with `veeamgo add job --template <Type>` or `--from` to create jobs rapidly.

##### Clone / Delete
```
veeamgo clone job --name "<Job Name>" --yes
veeamgo clone job --id <Job ID> --yes

veeamgo delete job --name "<Job Name>" --yes
veeamgo delete job --id <Job ID> --yes
```
- `clone` duplicates the selected job (keeping schedules disabled so you can adjust before activating).
- `delete` permanently removes the job; `--yes` is required to bypass the confirmation prompt.

##### Enable / Disable
```
veeamgo disable job "<Job Name>" --yes
veeamgo disable job --id <Job ID> --yes

veeamgo enable job "<Job Name>" --yes
veeamgo enable job --id <Job ID> --yes
```
`--yes` is required for safety; omit the name and use `--id` when multiple jobs share the same display name. The commands are idempotent and print the current state when no change is needed.

##### Start / Stop / Retry
```
veeamgo start job --name "<Job Name>" --yes [--active-full] [--start-chained]
veeamgo stop job --name "<Job Name>" --yes [--graceful=false] [--cancel-chained]
veeamgo retry job --name "<Job Name>" --yes [--start-chained]
```
- `start` launches a new session; optionally trigger an active full (`--active-full`) or cascade to chained jobs.
- `stop` requests a graceful stop by default; use `--graceful=false` for an immediate termination.
- `retry` reruns the most recent failed session and can trigger chained jobs when desired.
- All commands accept `--id` instead of `--name` and require `--yes` to execute.

#### Restore Points
```
veeamgo get restorepoint \
  --name "<Job Name>" \
  [--backup "<Backup Name>"] \
  [--restorepoint "<pattern>"] \
  [--object <Object ID>] \
  [--platform <Platform Name>] \
  [--platform-id <Platform ID>] \
  [--malware <Status>] \
  [--created-since <RFC3339>] \
  [--created-before <RFC3339>] \
  [--sort <column>] [--desc] \
  [--limit N]
```
`--backup` disambiguates jobs with multiple backups.

#### Replica Restore Points
```
veeamgo get replica \
  --name "<Replication Job>" \
  [--replica "<Replica Name>"] \
  [--replica-point "<pattern>"] \
  [--platform <Platform Name>] \
  [--platform-id <Platform ID>] \
  [--malware <Status>] \
  [--created-since <RFC3339>] \
  [--created-before <RFC3339>] \
  [--sort <column>] [--desc] \
  [--limit N]
```

#### Backup Files
```
veeamgo get backup files \
  --name "<Job Name>" \
  [--backup "<Backup Name>"] \
  [--file "<pattern>"] \
  [--gfs <Period>] \
  [--created-since <RFC3339>] \
  [--created-before <RFC3339>] \
  [--sort <column>] [--desc] \
  [--limit N]
```
Displays file size metrics and dedupe/compression ratios. The “GFS Periods” column is blank when none apply.

#### Backup Objects
```
veeamgo get objectsinbackup --name "<Job Name>" [--backup "<Backup Name>"]
```
Alias: `veeamgo get backup objects`. Lists VM/object entries and their restore point counts.

#### Sessions
```
veeamgo get session \
  [--name <pattern>] \
  [--job <Job ID>] \
  [--type <Type> ...] \
  [--state <State> ...] \
  [--result <Result> ...] \
  [--created-since <RFC3339>] [--created-before <RFC3339>] \
  [--ended-since <RFC3339>] [--ended-before <RFC3339>] \
  [--sort <column>] [--desc] \
  [--limit N]
```

##### Session Logs
```
veeamgo get session logs --id "<Session ID>" [--status <Status>]
```

##### Describe Session
```
veeamgo describe session --id "<Session ID>"
```

#### Tasks
```
veeamgo get task \
  [--name <pattern>] \
  [--session-id <Session ID>] \
  [--type <Task Type>] \
  [--session-type <Session Type>] \
  [--state <State>] [--result <Result>] \
  [--scan-type <Scan Type>] [--scan-result <Result>] [--scan-state <State>] \
  [--created-since <RFC3339>] [--created-before <RFC3339>] \
  [--ended-since <RFC3339>] [--ended-before <RFC3339>] \
  [--sort <column>] [--desc] \
  [--limit N]
```

Task logs mirror session logs:
```
veeamgo get task logs --id "<Task ID>" [--status <Status>]
```

Describe a task session:
```
veeamgo describe task --id "<Task ID>"
```

#### Inventory (Virtual Infrastructure)
```
veeamgo get inventory virtualinfra \
  [--platform vsphere|hyperv|clouddirector] \
  [--type <Type> ...] \
  [--name <pattern>] \
  [--host <pattern>] \
  [--limit N] [--skip N] \
  [--sort <field>] [--desc]
```

- `--platform` filters by platform family (VMware vSphere, Microsoft Hyper-V, VMware Cloud Director).
- `--type` narrows by server type (`VCenterServer`, `Scvmm`, `CloudDirectorServer`, etc.).
- `--name` and `--host` perform case-insensitive substring matches.

```
veeamgo get inventory virtualinfra objects \
  --name "<Server or Host>" \
  [--hierarchy HostsAndClusters|VmsAndTemplates|...] \
  [--object-type <Type> ...] \
  [--object-name <pattern>] \
  [--limit N] [--skip N] \
  [--sort <field>] [--desc]
```

##### Describe Virtual Infrastructure Host
```
veeamgo describe inventory virtualinfra \
  --name "<Server or Host Name>" \
  [--limit N] [--skip N] \
  [--sort <field>] [--desc]
```

Displays metadata for the specified inventory server (type, host, IDs, size).

#### Protection Groups
```
veeamgo get inventory protectiongroup \
  [--name <pattern>] \
  [--type <Type>] \
  [--limit N] \
  [--sort <field>] [--desc]
```

##### Protection Group Agents
```
veeamgo get inventory protectiongroup agents \
  --name "<Protection Group Name>" \
  [--limit N]
```

##### Describe Protection Group Agent
```
veeamgo describe inventory protectiongroup \
  --name "<Agent Name>" \
  [--group "<Protection Group Name>"]
```

Shows detailed agent metadata (state, versions, IP addresses, OS).

#### Security Analyzer
```
veeamgo start securityanalyzer --yes [--wait]
veeamgo get securityanalyzer results
veeamgo describe securityanalyzer schedule
```
- `start`: triggers an analyzer run; use `--wait` to block until the session finishes and the CLI prints the final status plus the latest analyzer results.
- `results`: lists the most recent compliance findings.
- `schedule`: shows how often the analyzer runs and notification settings.

#### Configuration Backup
```
veeamgo start configurationbackup --yes [--wait]
```
Runs an on-demand configuration backup, optionally waiting for completion to report the session result.

#### Traffic Rules & Exclusions
```
veeamgo get trafficrule
veeamgo get exclusionvm [--limit N]
```
```
veeamgo get trafficrule
veeamgo get exclusionvm [--limit N]
```
- `get trafficrule`: reviews preferred network rules and throttling settings.
- `get exclusionvm`: lists globally excluded VMs so you can audit skipped workloads.

#### WAN Accelerators
```
veeamgo get wanaccelerator \
  [--name <pattern>] \
  [--sort <column>] [--desc] \
  [--limit N]
```

Lists WAN accelerators alongside cache folder, traffic port, stream count, and high-bandwidth mode status to help validate WAN acceleration capacity.

##### Describe WAN Accelerator
```
veeamgo describe wanaccelerator --name "<Name>"
```

Returns a single-record table with host ID, cache sizing (unit-aware), stream count, and a YAML block of the full configuration payload.

#### General Options
```
veeamgo get generaloption [--output table|json]
```

Fetches global notification thresholds and SIEM integration toggles. Table output mirrors `describe` formatting so each flag renders as a key/value pair.

#### Malware Detection Events
```
veeamgo get malwaredetectionevent \
  [--type <Type>] \
  [--state <State>] \
  [--source <Source>] \
  [--severity <Severity>] \
  [--created-by <Account>] \
  [--engine <Engine>] \
  [--machine-name <Pattern>] \
  [--backup-object <ID>] \
  [--detected-since <RFC3339>] \
  [--detected-before <RFC3339>] \
  [--sort <column>] [--desc] \
  [--limit N]
```

Lists malware detection events (suspicious activity) raised by VBR. Table mode highlights severity, detection time, engine, and machine details. Use the detection timestamps to bracket events during an incident. Sorting defaults to ascending detection time; add `--desc` to reverse the order.

#### YARA Rules
```
veeamgo get yararule [--output table|json]
```

Displays uploaded YARA rule files so you can confirm which signatures are available to the malware scanner.

#### Licensing
```
veeamgo get license
veeamgo get license sockets [filters...]
veeamgo get license instances [filters...]
veeamgo get license capacity [filters...]
```

### 3.2 `describe` Commands
| Command | Purpose |
| --- | --- |
| `veeamgo describe managedserver --id <ID>` | Managed server details (type, status, description). |
| `veeamgo describe repository --name <Name>` | Repository inventory/config. |
| `veeamgo describe proxy --name <Name> [--type <Type>]` | Proxy details (transport, encryption, host info). |
| `veeamgo describe job <Name>` | Scheduling, repository, session summary. |
| `veeamgo describe session --id <ID>` | Detailed session view incl. YAML payload. |
| `veeamgo describe task --id <ID>` | Task session detail with raw payload. |
| `veeamgo describe configurationbackup` | Configuration backup policy. |
| `veeamgo describe inventory virtualinfra --name <Server>` | Metadata for a virtual infrastructure server. |
| `veeamgo describe inventory protectiongroup --name <Agent> [--group <Group>]` | Protection group agent details. |
| `veeamgo describe exclusionvm <ID>` | Global VM exclusion entry. |
| `veeamgo describe security analyzer schedule` | Security analyzer scheduling & notifications. |

### 3.3 `rescan` Commands
```
veeamgo rescan repository [--id <ID> ... | --all] [--wait]
veeamgo rescan server [--id <ServerID> | --all] [--wait]
```

---

## 4. JSON Output

`--output json` prints the API payload verbatim (including pagination metadata). Useful for:
- Scripting (`jq`, pipelines).
- Audit snapshots.
- Debugging differences between VBR versions.

Remember: table output is optimized for humans; JSON output is for tooling.

---

## 5. Error Handling

- Missing required flags throw messages via `requireFlag`, e.g.:
  ```
  error: flag --name is required (provide the backup job name for restore points)
  ```
- The CLI sets `SilenceUsage=true`, so usage text is suppressed; the smoke script validates error paths explicitly.

---

## 6. Automation & Smoke Testing

The project ships with `scripts/veeamgo_e2e.sh`, which:
1. Runs table-mode commands (`get` + `describe`).
2. Repeats the same endpoints in JSON mode.
3. Executes negative tests to confirm consistent flag handling.

`scripts/veeamgo_e2e.sh` expects `veeamgo` and `jq` in `PATH`. Customize via environment:

- `LOG_FILE=<path>` – where to store logs (default `veeamgo-test-YYYYMMDD-HHMMSS.log`).
- `RUN_MUTATING_COMMANDS=1` – placeholder for future write tests (currently no mutating commands run).

---

## 7. Development Notes

- Go version ≥ 1.19, standard Go toolchain (`gofmt`, `goimports`, `golangci-lint`).
- Timestamps must use `formatTimestamp`, `formatTimestampValue`, or `formatAPITime` helpers to respect the server timezone.
- Required flag validation should call `requireFlag` instead of relying on Cobra’s `MarkFlagRequired`.
- Table commands should always render header rows even when the data set is empty (handled automatically by `pkg/output`).
- Run unit tests with:
  ```
  go test ./...
  ```

---

## 8. Examples

```bash
# Discover repositories and show details
veeamgo login --server vbr.lab.local --username svc-veeam --save-password
veeamgo get repository --limit 10
veeamgo describe repository --name "Default Backup Repository"

# Drill into a job’s restore points and replica history
veeamgo get job --name "Replication"
veeamgo get restorepoint --name "Replication Job" --limit 5 --output json | jq '.restorePoints[] | {name, creationTime}'
veeamgo get replica --name "Replication Job" --limit 5

# Track licensing
veeamgo get license sockets --sort name --limit 20
veeamgo get license instances --type WindowsAgent --limit 10 --output json

# Review security analyzer configuration/results
veeamgo describe securityanalyzer schedule
veeamgo get securityanalyzer results --output json | jq '.items[] | {practice, status}'
```

---

## 9. Release Artifacts

Builds are typically produced with CGO disabled and cached modules:

```bash
CGO_ENABLED=0 GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache \
  GOOS=darwin  GOARCH=amd64 go build -o dist/veeamgo-darwin-amd64 .
# repeat for darwin/arm64, linux/amd64, linux/arm64, windows/amd64
```

---

## 10. Support & Contributions

At this stage the CLI is focused on read-only inspection. Feature requests and bug reports can be submitted via GitHub issues. When contributing:
- Include unit tests for new helpers or API integrations.
- Update `docs/user-guide.md` and the smoke script if the CLI surface changes.
- Ensure `go test ./...` passes before opening a PR.

Happy auditing! 💚
