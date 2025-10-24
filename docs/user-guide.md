# VeeamGo User Guide

*Updated: 2025‑10‑18*



[TOC]

---

## 1. CLI Overview

`veeamgo` is a command line client for Veeam Backup & Replication (VBR). It exposes inventory and status information via a verb‑first grammar inspired by `kubectl`:

```
veeamgo <verb> <resource> [subcommand] [flags]
```

- **Primary verbs**: `get`, `describe`, `rescan`, `add`, `edit`, `enable`, `disable`, `clone`, `delete`, `start`, `stop`, `retry`, `template`
- **Global flags** (available everywhere):
  | Flag | Description |
  | --- | --- |
  | `--config <path>` | Override config file (default `~/.veeamgo/config.yaml`). |
  | `--profile <name>` | Select an existing profile; defaults to the active profile. |
  | `--output table|json` | Human‑readable tables (`table`, default) or raw API payloads (`json`). |



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
- When the profile does not specify an API revision, the CLI makes a one-time `serverInfo` request to negotiate the highest supported revision and then caches it. If you already know the correct revision (for example `1.3-rev0`), pass `--api-version 1.3-rev0` or set `VEEAMGO_API_VERSION=1.3-rev0` to skip the negotiation round-trip.

### Logout
```
veeamgo logout          # Clears the active profile session
veeamgo logout --all    # Removes all cached sessions
```

---

## 3. Command Reference

### 3.1 `get` Commands (Table + JSON)

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
  --description "Repository gateway" \
  --credentials-storage Permanent \
  --ssh-fingerprint "ssh-rsa 3072 AAAAB3NzaC1yc2EAAAADAQABAAABAQ..." \
  [--credentials-id <UUID>] \
  [--handshake-code 123456] \
  [--single-use-username repo --single-use-password ***** ...] \
  [--wait] \
  [--yes]
```
- If `--description` is omitted, the CLI generates a friendly description with the creation date.
- Provide `--credentials-id` to reuse an existing record; otherwise the CLI creates standard credentials from `--username`/`--password`.
- Leave `--password` empty to be prompted securely.
- `--wait` blocks until the provisioning session finishes; otherwise the command prints the session ID for follow-up (`veeamgo describe session --id <ID>`).
- `--connect-mode` defaults to `Credential`, which uses a stored account (`--username`/`--password` or `--credentials-id`).
- Set `--connect-mode Certificate` when the host has a deployment kit installed; in that mode omit all credential flags and the CLI submits the certificate-based request automatically.
- For Linux, the CLI forwards flags directly to the API. Provide a fingerprint with `--ssh-fingerprint`; add `--handshake-code` when re-adding factory appliances that require pairing, and include any single-use credential fields explicitly.

##### Rescan
```
veeamgo rescan managedserver \
  [--id <Managed Server ID> | --name "<Managed Server Name>" | --all] \
  [--wait]
```
- Use `--all` to refresh every managed server registered with the backup infrastructure. Do not combine `--all` with `--id` or `--name`.
- For single hosts, provide either `--id` or `--name`. When a name is supplied the CLI resolves it to an ID before issuing the request (duplicates are rejected).
- Each invocation creates a rescan session; without `--wait` the CLI prints the session ID so you can track progress (`veeamgo get session logs --id <ID>`).
- Add `--wait` to block until the session completes. The CLI prints the final result (`Success`, `Failed`, etc.) and the platform-supplied message, if any.

##### Delete
```
veeamgo delete managedserver \
  [--id <Managed Server ID> | --name "<Managed Server Name>"] \
  [--wait] \
  [--yes]
```
- Provide either `--id` or `--name` (not both). When only the name is supplied, the CLI resolves it to a single managed server before proceeding.
- The command always creates an infrastructure deletion session; without `--wait` the CLI prints the session ID so you can track status (`veeamgo get session logs --id <ID>`).
- Add `--wait` to block until the session finishes — upon completion the CLI prints the final result (Success, Failed, Warning) and any descriptive message returned by the platform.
- `--yes` skips the safety prompt. Use it for non-interactive scripts after validating the target.

#### Repository
```
veeamgo get repository [--type <Type>] [--name <pattern>] [--limit N]
```
Lists repositories with capacity/free/used metrics. By default every repository type (local, hardened, SMB/NFS, object storage) is returned; add `--type` one or more times to narrow the query.

##### Describe
```
veeamgo describe repository --name "<Repository Name>"
```
Detailed view including raw YAML configuration.

##### Add
```
veeamgo add repository \
  --type WinLocal|LinuxLocal|LinuxHardened|SMB|NFS \
  --name "<Repository Name>" \
  [--description "<Text>"] \
  [type-specific flags] \
  [--max-tasks N] \
  [--read-write-limit MBps] \
  [--import-backup[=true|false]] \
  [--import-index[=true|false]] \
  [--disable] \
  [--wait] \
  [--yes]
```
- **WinLocal / LinuxLocal**: require `--host <Managed Server>` and `--path <Folder>`. The mount server defaults to the same host but can be overridden with `--mount-server`.
- **LinuxHardened**: same as LinuxLocal with optional `--fast-clone` and `--immutable-days <N>` to control XFS fast cloning and immutability retention.
- **SMB**: require `--share-path //<server>/<share>` and `--credentials-id <UUID>`. Use `--gateway-server` (repeatable) and/or `--gateway-auto=false` to control gateway placement. Mount server must be specified (`--mount-server`).
- **NFS**: require `--share-path nfs://<server>/<export>`. Gateway settings mirror SMB and a mount server is required.
- `--max-tasks` sets the repository task limit when greater than zero. `--read-write-limit` caps throughput in MB/s; supply `0` (default) to leave unlimited. `--wait` blocks until the provisioning session completes; otherwise the command prints the session ID.

##### Edit
```
veeamgo edit repository \
  [--name "<Repository Name>" | --id <Repository ID>] \
  [--type WinLocal|LinuxLocal|LinuxHardened|SMB|NFS] \
  [--new-name "<New Name>"] \
  [--description "<Text>"] \
  [type-specific flags] \
  [--max-tasks N] \
  [--read-write-limit MBps] \
  [--fast-clone[=true|false]] \
  [--immutable-days N] \
  [--import-backup[=true|false]] \
  [--import-index[=true|false]] \
  [--disable[=true|false]] \
  [--wait] \
  [--yes]
```
- Identify the repository with either `--name` (optionally constrained by `--type`) or `--id`. Duplicated names produce suggestions.
- Local repositories accept updates to `--host`, `--path`, and mount server settings; the CLI preserves existing values when flags are omitted.
- Linux hardened repositories support `--fast-clone` and `--immutable-days` toggles. SMB/NFS repositories can switch shares (`--share-path`), credentials (`--credentials-id`), gateways (`--gateway-auto`, `--gateway-server`), and mount server settings.
- Use `--max-tasks`/`--read-write-limit` to rewrite performance caps (set to `0` to disable). `--disable` toggles repository availability. Combine with `--wait` to follow the configuration session synchronously.

#### Object Repository
```
veeamgo get objectrepository [--type <Type>] [--name <pattern>] [--limit N]
```
Lists object storage repositories (Amazon S3, S3 compatible, Azure Blob, etc.).

##### Describe
```
veeamgo describe objectrepository --name "<Repository Name>"
```
Shows the raw object storage configuration payload.

##### Add
```
veeamgo add objectrepository types
```
- Lists every supported object storage type, the canonical identifier used by the REST API, and whether a scaffolded subcommand is available.

###### Provider-aware shortcuts
```
veeamgo add objectrepository amazon-s3 \
  --name "<Repository Name>" \
  --credentials-id <UUID> \
  --bucket-name "<Bucket>" \
  --folder "<Prefix>" \
  --region-id "<aws-region>" \
  --mount-server-id <UUID> \
  --mount-server-cache "D:\VeeamCache" \
  [--region-scope Global|China|Government] \
  [--connection-type Direct|SelectedGateway] \
  [--gateway-id <UUID> ...] \
  [--immutability-enabled] \
  [--immutability-days N] \
  [--description "<Text>"] \
  [--disable] \
  [--wait] \
  [--yes]
```
- `amazon-s3`, `s3-compatible`, `wasabi-cloud`, `azure-blob`, `azure-archive`, and `veeam-data-cloud-vault` subcommands expose the common fields for each provider so you do not have to hand-craft JSON payloads. Optional parameters fall back to reasonable defaults (`Direct` connectivity, Windows mount server with vPower enabled, immutability disabled unless requested).
- Azure Archive requires a proxy appliance definition. The helper exposes the key fields (`--proxy-subscription-id`, `--proxy-resource-group`, `--proxy-virtual-network`, `--proxy-subnet`, `--proxy-instance-size`, `--proxy-redirector-port`) so you can stitch the payload together without editing JSON manually.
- Wasabi builds on the S3-compatible flags but constrains the schema to match the REST contract (region, bucket, folder, optional immutability).
- `--spec` remains available on these subcommands: supply an existing template and use flags to patch particular fields. The CLI merges flag values into the loaded payload before submitting it.
- Use `--set` for rare or advanced fields (for example proxy appliance details) — the helper flags simply write to the same nested paths (`bucket.immutability.isEnabled`, `account.connectionSettings.gatewayServerIds`, and so on).

###### Raw JSON workflow
```
veeamgo add objectrepository \
  --type <Object Repository Type> \
  --spec ./object-repo.json \
  [--name "<Repository Name>"] \
  [--description "<Text>"] \
  [--set path.to.field=value ...] \
  [--disable] \
  [--wait] \
  [--yes]
```
- `--spec` must reference a JSON document conforming to the REST schema for the chosen repository (`AmazonS3StorageSpec`, `AzureBlobStorageSpec`, `S3CompatibleStorageSpec`, ...). Use `-` to read from stdin.
- `--set` accepts dot-notation overrides (for example `--set bucket.bucketName=my-bucket`). Values support booleans (`true`/`false`) and numbers in addition to strings.
- `--name`, `--description`, and `--disable` override the corresponding top-level fields from the spec. All other configuration (credentials, bucket/container details, connection mode) should be provided by the JSON file.
- `--wait` blocks until the provisioning session completes; otherwise the command prints the session ID for follow-up.

##### Edit
```
veeamgo edit objectrepository \
  [--name "<Repository Name>" | --id <Repository ID>] \
  [--type <Object Repository Type>] \
  [--spec ./patched-object-repo.json] \
  [--set path.to.field=value ...] \
  [--new-name "<New Name>"] \
  [--description "<Text>"] \
  [--disable] \
  [--wait] \
  [--yes]
```
- If `--spec` is omitted the CLI fetches the current repository payload and applies overrides in place. Provide a spec to replace the configuration with a file-based template.
- `--set` uses the same dot-notation override format as the add command; use it to tweak endpoints, credentials IDs, immutability, and other nested settings without rewriting the entire spec.
- `--type` can be supplied to validate the expected repository type when repository names are duplicated.

##### Rescan
```
veeamgo rescan objectrepository [--id <ID> ... | --name "<Name>" ... | --all] [--wait]
```
- Refreshes metadata for object repositories. The CLI rejects attempts to rescan non-object repositories.

##### Delete
```
veeamgo delete objectrepository \
  [--name "<Repository Name>" | --id <Repository ID>] \
  [--delete-backups] \
  [--yes]
```
- Removes an object storage repository. Use `--delete-backups` to remove backup data from the external bucket/container as part of the deletion.

#### Proxy
```
veeamgo get proxy [--name <pattern>] [--type vmware|hyperv|general] [--host-id <ID>] [--sort <column>] [--desc] [--limit N]
```

##### Describe
```
veeamgo describe proxy --name "<Proxy Name>" [--type vmware|hyperv|general]
```

##### Add
```
veeamgo add proxy \
  --managed-server "<Managed Server Hostname or IP>" \
  [--name "<Proxy Name>"] \
  [--type vmware|hyperv|general] \
  [--transport-mode auto|directAccess|virtualAppliance|network] \
  [--max-tasks N] \
  [--failover-to-network] \
  [--host-to-proxy-encryption] \
  [--auto-select-datastores] \
  [--wait] \
  [--yes]
```
- `--managed-server` is required and must reference a server that already exists in the managed server inventory (`veeamgo add managedserver ...`).
- `--name` defaults to the managed server name/IP so you only provide a value when you need a different proxy label.
- The CLI rejects duplicates when a proxy with the same name or `(type, managed server)` pair already exists.
- `--type` accepts `vmware`, `hyperv`, or `general`, which map to the VBR API types `ViProxy`, `HvProxy`, and `GeneralPurposeProxy` respectively.
- `--transport-mode`, `--failover-to-network`, `--host-to-proxy-encryption`, and `--auto-select-datastores` apply to VMware proxies (API type `ViProxy`); Hyper-V (`HvProxy`) and general (`GeneralPurposeProxy`) proxies ignore them.
- Use `--wait` to block until the provisioning session completes; without it the command prints the session ID for follow-up via `veeamgo get session`.

##### Delete
```
veeamgo delete proxy \
  [--name "<Proxy Name>"] \
  [--id <Proxy ID>] \
  [--type vmware|hyperv|general] \
  [--yes]
```
- Provide either `--name` (optionally with `--type` to disambiguate duplicates) or `--id`.
- Without `--yes` the CLI prompts for confirmation and can be cancelled safely.
- The command validates that the proxy exists before issuing the delete request and surfaces the session-friendly identifier in the confirmation message.

##### Enable / Disable
```
veeamgo enable proxy \
  [--name "<Proxy Name>"] \
  [--id <Proxy ID>] \
  [--type vmware|hyperv|general] \
  [--yes]

veeamgo disable proxy \
  [--name "<Proxy Name>"] \
  [--id <Proxy ID>] \
  [--type vmware|hyperv|general] \
  [--yes]
```
- Supply either `--name` or `--id`; combine `--type` with `--name` when duplicate proxy names exist.
- When `--yes` is omitted the CLI asks for confirmation before changing the state.
- If the proxy is already in the requested state the command exits gracefully without calling the API.

##### Edit
```
veeamgo edit proxy \
  [--name "<Proxy Name>" | --id <Proxy ID>] \
  [--type vmware|hyperv|general] \
  [--new-name "<New Name>"] \
  [--description "<Text>"] \
  [--managed-server "<Managed Server Host>"] \
  [--max-tasks N] \
  [--transport-mode auto|directAccess|virtualAppliance|network] \
  [--failover-to-network=<true|false>] \
  [--host-to-proxy-encryption=<true|false>] \
  [--auto-select-datastores | --no-auto-select-datastores] \
  [--wait] \
  [--yes]
```
- Supply either `--name` (optionally constrained with `--type`) or `--id` to identify the proxy.
- Flags mirror those available during creation; only the values provided are changed. Untouched settings are preserved.
- `--managed-server` lets you re-point the proxy to another managed server that already exists in inventory.
- Use `--auto-select-datastores` / `--no-auto-select-datastores` to toggle datastore discovery for VMware proxies; the CLI retains the current datastore list when switching back to manual mode.
- `--wait` blocks until the infrastructure session reporting the edit completes, otherwise the command prints the session ID so you can monitor progress later.

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
veeamgo describe job --name "<Job Name>"
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

#### Backups
```
veeamgo get backup list \
  [--name "<Job Name>"] \
  [--job-id <Job ID>] \
  [--job-type <EJobType>] \
  [--created-since <RFC3339>] \
  [--created-before <RFC3339>] \
  [--sort <column>] [--desc] \
  [--limit N]
```
The `list` subcommand returns all backups that match the filters. Use `--output json` to capture the raw payload for automation or dashboards. When only a subset of backups is required you can combine `--name`, `--job-id`, and `--job-type`.

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

##### Disk Publishing
```
veeamgo start publishdisk \
  --restore-point-id <Restore Point ID> \
  --allowed-ip <IP Address> [--allowed-ip <IP Address> ...] \
  [--disk "Disk 1" --disk "Disk 2" ...] \
  [--wait]
```
- Always supply the restore point identifier gathered from `veeamgo get restorepoint` (the table now includes the `Id` column).
- Provide at least one `--allowed-ip` value. Repeat the flag for every initiator that must connect to the iSCSI target.
- `--disk` is optional; omit it to publish every disk in the restore point. Repeat the flag to enumerate specific disks.
- The command currently publishes disks in iSCSI target mode only. Additional publishing modes will be surfaced once the underlying API gaps are resolved.
- The CLI waits briefly (up to about two minutes) for the publishing session to complete and prints the mount details automatically. Use `--wait` for longer operations or to block until completion.
- Use `--wait` when you want the CLI to block until the publishing session finishes. Without it the command prints the session ID so you can monitor progress (`veeamgo get session logs --id <Session ID>`).
- The CLI prints platform messages (success or failure) after the session finishes so you can confirm the disks are ready or address environmental issues quickly.

##### Stop Published Disks
```
veeamgo stop publishdisk \
  --id <Mount ID> [--id <Mount ID> ...] \
  [--wait]
```
- Provide one or more mount identifiers (returned by `veeamgo start publishdisk --wait` or `veeamgo get publisheddisk`).
- Without `--wait` the CLI starts the unpublish session and prints the session ID for follow-up; add `--wait` to block until the disks are detached and view the final status message.
- Errors are reported per mount so scripts can react if any disk fails to unpublish.

##### List Published Disks
```
veeamgo get publisheddisk --all [--limit N]
veeamgo get publisheddisk --id <Mount ID>
```
- Use `--all` to enumerate active disk publishing sessions (FUSE and iSCSI). Add `--limit` to cap the number of rows when scripting.
- Supply `--id` (the mount identifier returned by `veeamgo start publishdisk --wait`) to inspect a specific mount, including target host, IP addresses, and port assignment.
- Table output summarises each mount; switch to JSON (`--output json`) to retrieve the full REST payload.

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
veeamgo get backup objects \
  --name "<Job Name>" \
  [--backup "<Backup Name>"] \
  [--platform <Platform Name>] \
  [--limit N]
```
Lists VM/object entries and their restore point counts.

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
| `veeamgo describe proxy --name <Name> [--type vmware|hyperv|general]` | Proxy details (transport, encryption, host info). |
| `veeamgo describe job --name <Name>` | Scheduling, repository, session summary. |
| `veeamgo describe session --id <ID>` | Detailed session view incl. YAML payload. |
| `veeamgo describe task --id <ID>` | Task session detail with raw payload. |
| `veeamgo describe configurationbackup` | Configuration backup policy. |
| `veeamgo describe inventory virtualinfra --name <Server>` | Metadata for a virtual infrastructure server. |
| `veeamgo describe inventory protectiongroup --name <Agent> [--group <Group>]` | Protection group agent details. |
| `veeamgo describe exclusionvm <ID>` | Global VM exclusion entry. |
| `veeamgo describe security analyzer schedule` | Security analyzer scheduling & notifications. |

### 3.3 `rescan` Commands
```
veeamgo rescan repository [--id <ID> ... | --name "<Name>" ... | --all] [--wait]
veeamgo rescan objectrepository [--id <ID> ... | --name "<Name>" ... | --all] [--wait]
veeamgo rescan server [--id <ServerID> | --all] [--wait]
```
- For repository/objectrepository commands mix `--id` and `--name` as needed; duplicates are ignored. `--all` cannot be combined with either option.

### 3.4 `delete` Commands
```
veeamgo delete repository [--name "<Name>" | --id <ID>] [--delete-backups] [--yes]
veeamgo delete objectrepository [--name "<Name>" | --id <ID>] [--delete-backups] [--yes]
veeamgo delete proxy [--name "<Name>" | --id <ID>] [--type <Type>] [--yes]
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

## 6. Examples

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
