# VeeamGo User Guide

English | [中文指南](UserGuide.zh-CN.md)

## Audience & Scope
This guide explains how to authenticate and operate the VeeamGo CLI against a Veeam Backup & Replication (VBR) server. It focuses on day-to-day operator workflows, infrastructure management, and the analytic tooling shipped with the CLI. Every command assumes you already logged in successfully and that your VBR account has the necessary permissions.

## 1. CLI Basics

### 1.1 Verb-First Syntax
The CLI follows the pattern `veeamgo <verb> <resource> [subcommand] [flags]`. Examples:
```bash
veeamgo get job --limit 10
veeamgo describe repository --name "Default Backup Repository"
veeamgo start job --name "Daily Backup"
```

### 1.2 Output Formats
Use `--output table` (default) for human-readable tables or `--output json` to see the raw REST payload. JSON mode is useful when piping data to tooling such as `jq`.

### 1.3 Global Flags
| Flag | Description |
| --- | --- |
| `--config <path>` | Override the configuration file location (default `~/.veeamgo/config.yaml`). |
| `--profile <name>` | Select or create a profile; defaults to the current active profile. |
| `--output table|json` | Choose table or JSON rendering for the current invocation. |
| `--api-version <rev>` | Force a specific REST API revision instead of auto-negotiation. |
| `--help` | Available on every command for exhaustive flag descriptions. |

### 1.4 Filters, Sorting, and Selectors
- Name filters accept `*` wildcards (`--name *prod*`).
- Most list commands accept `--limit`, `--sort`, and `--desc` for pagination and ordering.
- Timestamp filters (`--created-since`, `--detected-before`, etc.) expect RFC3339 values (e.g. `2024-05-01T00:00:00Z`).
- Many commands accept both `--name` and `--id`; specify one to avoid ambiguity.
- Some commands accept repeatable flags such as `--type`, `--gateway-id`, or `--disk`. Pass them multiple times to target several values.
- Blueprint-oriented commands (`veeamgo add|edit job`, `veeamgo add objectrepository`) support `--set key=value` overrides without editing JSON by hand.

### 1.5 Sessions, Prompts, and Safety Flags
Long-running operations return a session ID. Add `--wait` to follow the session until it completes, or inspect it later with `veeamgo get session logs --id <uuid>`. Mutating commands prompt for confirmation; use `--yes` to skip prompts in automation. When the CLI prints a session ID, it is safe to rerun the command—it will not create duplicate resources.

## 2. Authentication & Profiles

### 2.1 Login
Authenticate with `veeamgo login`. Key flags:

| Flag | Purpose |
| --- | --- |
| `--server <host|url>` | REST API base URL or host name (HTTPS enforced; default port 9419). |
| `--port <n>` | Override the REST port when only a host name is supplied. |
| `--username <user>` | VBR account used for authentication. |
| `--password <pass>` | Optional inline password; omit to be prompted securely. |
| `--scope <scopes>` | Additional OAuth scopes when required by the server. |
| `--insecure` | Skip TLS validation (lab use only). |
| `--save-password` | Persist the password encrypted in the config for silent logins. |
| `--set-default` | Make the current profile the default for future commands. |

Environment variables:
- `VEEAMGO_PASSWORD` supplies the password non-interactively.
- `VEEAMGO_API_VERSION` forces a specific API revision when login is finished.

Upon success the CLI writes `~/.veeamgo/config.yaml` and caches session tokens in `~/.veeamgo/sessions.json`.

### 2.2 Logout
- `veeamgo logout` clears the active profile session.
- `veeamgo logout --all` removes every cached session (recommended before switching labs or sharing a workstation).

### 2.3 Profile Storage & Switching
Profiles are keyed by name. Use `--profile qa` with any command to operate against that profile. If the profile does not exist, login will create it. You can edit the YAML file manually when rotating credentials, but always keep file permissions restricted (`0700` directory, `0600` file).

## 3. Command Reference

Unless stated, commands honour the global flags described above. Examples assume table output for readability.

### 3.1 Server & Platform Configuration
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get server info` | Display server platform, build, database, and patch information. | None |
| `veeamgo get server time` | Show the server clock and time zone. | None |
| `veeamgo get generaloption` | Inspect global notification, SIEM, and alert settings. | None |
| `veeamgo describe configurationbackup` | Review scheduled configuration backup settings. | None |
| `veeamgo start configurationbackup [--yes] [--wait]` | Trigger an on-demand configuration backup and optionally wait for completion. | `--yes`, `--wait` |
| `veeamgo license summary` | Summarise license owner, type, and expiration dates. | None |
| `veeamgo license sockets [--limit N]` | List socket-based workload consumption. | `--limit` |
| `veeamgo license instances [--limit N]` | List instance usage and revocation eligibility. | `--limit` |
| `veeamgo license capacity [--limit N]` | Display capacity-tier usage details. | `--limit` |

### 3.2 Infrastructure & Inventory

#### Managed Servers
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get managedserver [--type WindowsHost] [--name *prod*] [--limit 50]` | List managed servers with optional type and name filters. | `--type`, `--name`, `--limit` |
| `veeamgo describe managedserver --id <uuid>` | Show details for a specific managed server. | `--id` |
| `veeamgo add managedserver vsphere --name vc01.lab --username svc --password ***** [--thumbprint sha1] [--wait] [--yes]` | Register a vCenter/ESXi host. Supports `--credentials-id` to reuse stored credentials and `--port` to override HTTPS port. | `--name`, credential flags, `--port`, `--thumbprint`, `--wait`, `--yes` |
| `veeamgo add managedserver windows --name repo01.lab --connect-mode Credential|Certificate --username admin ...` | Add a Windows host. Supports certificate mode and credential reuse. | `--name`, `--connect-mode`, credential flags, `--wait`, `--yes` |
| `veeamgo add managedserver linux --name repo01.lab --connect-mode Credential|SingleUse|Certificate ...` | Add a Linux host. Supports automatic fingerprinting via `--ssh-fingerprint` and single-use credentials (`--single-use-*` flags). | `--name`, `--connect-mode`, credential flags, `--ssh-fingerprint`, `--handshake-code`, `--single-use-*`, `--wait`, `--yes` |
| `veeamgo rescan managedserver [--all | --id <uuid> | --name host] [--wait]` | Refresh managed server inventory data. | `--all`, `--id`, `--name`, `--wait` |
| `veeamgo delete managedserver --id <uuid> [--wait] [--yes]` | Remove a managed server. A name can be supplied instead of `--id`. | `--id`/`--name`, `--wait`, `--yes` |

#### Backup Repositories
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get repository [--type WinLocal] [--name *repo*] [--limit 20]` | List repositories and capacity metrics. | `--type`, `--name`, `--limit` |
| `veeamgo describe repository --name "Default Backup Repository"` | Show repository configuration and raw YAML. | `--name` |
| `veeamgo add repository winlocal --host repo01.lab --path D:\Backups --mount-server repo01.lab [--max-tasks 4] [--wait] [--yes]` | Add a Windows-based repository. | `--host`, `--path`, `--mount-server`, `--max-tasks`, `--read-write-limit`, `--import-backup`, `--import-index`, `--disable`, `--wait`, `--yes` |
| `veeamgo add repository linuxlocal --host repo01.lab --path /backups ...` | Add a Linux repository using SSH-managed hosts. | Same flags as Windows repositories |
| `veeamgo add repository linuxhardened --host repo01.lab --path /immutable --immutable-days 14 [--fast-clone] ...` | Add an immutable hardened repository. | `--immutable-days`, `--fast-clone`, plus common flags |
| `veeamgo add repository smb|nfs --share-path //<server>/<share> --credentials-id <uuid> --mount-server repo01.lab [--gateway-server id] ...` | Add SMB or NFS repositories with gateway control. | `--share-path`, `--credentials-id`, `--gateway-auto`, `--gateway-server`, `--mount-server`, common flags |
| `veeamgo edit repository --name Repo1 [--new-name Repo2] [--max-tasks 8] ...` | Update repository settings without rebuilding JSON by hand. | `--name`/`--id`, `--type`, tuning flags, `--wait`, `--yes` |
| `veeamgo rescan repository [--name Repo1 | --id <uuid> | --all] [--wait]` | Validate repository configuration with a rescan session. | `--all`, `--name`, `--id`, `--wait` |
| `veeamgo delete repository --name Repo1 [--wait] [--yes]` | Remove a repository and associated sessions. | `--name`/`--id`, `--wait`, `--yes` |

#### Object Storage Repositories
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get objectrepository [--type AmazonS3] [--name *vault*] [--limit 20]` | List object storage repositories. | `--type`, `--name`, `--limit` |
| `veeamgo describe objectrepository --name "S3 Offsite"` | Show detailed configuration and placement policy. | `--name` |
| `veeamgo add objectrepository amazon-s3 ...` | Scaffolds an Amazon S3 repository without hand-editing JSON. | `--name`, `--description`, `--credentials-id`, `--bucket-name`, `--folder`, `--region-id`, `--region-scope`, `--connection-type`, `--gateway-id`, `--mount-server-id`, `--mount-server-cache`, `--mount-server-vpower`, `--immutability-enabled`, `--immutability-days`, `--spec`, `--set`, `--disable`, `--wait`, `--yes` |
| `veeamgo add objectrepository s3-compatible|wasabi-cloud|azure-blob|azure-archive|veeam-data-cloud-vault ...` | Provider-specific helpers with provider-appropriate flags. | See `--help` for each provider |
| `veeamgo add objectrepository types` | List supported provider identifiers and whether a scaffold command exists. | None |
| `veeamgo edit objectrepository --name "S3 Offsite" [--type AmazonS3] [--set account.bucketName=new-bucket] ...` | Apply updates to existing object repositories. | `--name`/`--id`, `--type`, `--set`, `--spec`, `--disable`, `--wait`, `--yes` |
| `veeamgo rescan objectrepository [--name Repo | --id <uuid> | --all] [--wait]` | Rescan object storage repositories to validate access. | `--name`/`--id`, `--all`, `--wait` |
| `veeamgo delete objectrepository --name "S3 Offsite" [--wait] [--yes]` | Delete an object repository. | `--name`/`--id`, `--wait`, `--yes` |

#### Scale-Out Backup Repositories (SOBR)
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get sobr [--name *prod*] [--limit 20]` | List scale-out repositories with placement policies. | `--name`, `--limit` |
| `veeamgo describe sobr --name "SOBR01"` | Show placement, capacity/archival tiers, and YAML config. | `--name` |

#### WAN Accelerators
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get wanaccelerator [--name *source*] [--limit 20]` | List WAN accelerators and cache settings. | `--name`, `--limit` |
| `veeamgo describe wanaccelerator --name "WAN-A"` | Show cache folder, port, and configuration. | `--name` |

#### Proxies
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get proxy list [--type vmware] [--name *proxy*] [--sort name] [--limit 20]` | List proxies with platform, status, and capacity. | `--type`, `--name`, `--sort`, `--desc`, `--limit` |
| `veeamgo describe proxy --name Proxy01 [--type vmware]` | Show proxy configuration and transport mode. | `--name`, `--type` |
| `veeamgo add proxy --managed-server host01.lab --type vmware --transport-mode auto [--max-tasks 4] [--yes] [--wait]` | Provision a new proxy on an existing managed server. | `--managed-server`, `--type`, `--max-tasks`, VMware-specific flags, `--wait`, `--yes` |
| `veeamgo edit proxy --name Proxy01 [--new-name Proxy02] [--managed-server host02] ...` | Update proxy metadata and capacity limits. | `--name`/`--id`, tuning flags, `--wait`, `--yes` |
| `veeamgo delete proxy --name Proxy01 [--wait] [--yes]` | Remove a proxy. | `--name`/`--id`, `--wait`, `--yes` |
| `veeamgo enable proxy --name Proxy01 [--yes] [--wait]` | Enable a disabled proxy. | `--name`/`--id`, `--yes`, `--wait` |
| `veeamgo disable proxy --name Proxy01 [--yes] [--wait]` | Disable a proxy without deleting it. | `--name`/`--id`, `--yes`, `--wait` |

#### Traffic Rules & Global Exclusions
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get trafficrule` | List traffic throttling/encryption rules. | None |
| `veeamgo get exclusionvm [--sort id] [--limit 50]` | Show globally excluded VMs. | `--sort`, `--desc`, `--limit` |
| `veeamgo describe exclusionvm <id>` | Inspect a specific global exclusion entry. | Positional `id` argument |

#### Inventory Browsing
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get inventory virtualinfra [--platform vsphere] [--type VCenterServer] [--name *prod*] [--limit 200]` | List virtual infrastructure servers (vSphere, Hyper-V, Cloud Director). | `--platform`, `--type`, `--name`, `--host`, `--limit`, `--sort`, `--desc` |
| `veeamgo get inventory virtualinfra objects --name vc01.lab [--hierarchy HostsAndClusters] [--object-type VirtualMachine]` | Browse inventory objects beneath a server. | `--name`, `--hierarchy`, `--object-type`, `--object-name`, `--limit`, `--sort`, `--desc` |
| `veeamgo describe inventory virtualinfra --name vc01.lab` | Show metadata (URNs, IDs) for a given inventory server. | `--name`, `--limit`, `--sort`, `--desc` |
| `veeamgo get inventory unstructured [--type NasServer] [--limit 200]` | List NAS/file-share inventory. | `--type`, `--name`, `--limit`, `--sort`, `--desc` |
| `veeamgo describe inventory unstructured --id <uuid>` | Inspect a specific unstructured inventory item. | `--id` |
| `veeamgo get inventory protectiongroup [--type Agent] [--name *prod*] [--limit 200]` | List protection groups and their scopes. | `--type`, `--name`, `--limit`, `--sort`, `--desc` |
| `veeamgo describe inventory protectiongroup --id <uuid>` | Show configuration for a protection group. | `--id` |

#### Certificate Fingerprints
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get fingerprint --server host --type LinuxHost [--credentials-storage Certificate] [--handshake-code 123456] [--port 22]` | Retrieve connection fingerprints before adding managed servers. | `--server`, `--type`, `--credentials-storage`, `--credentials`/`--handshake-code`, `--port` |

### 3.3 Jobs, Backups, and Restore Data

#### Jobs
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get job [--type Backup] [--status Running] [--name *sql*] [--limit 50]` | List jobs with status, schedule, and repository. | `--type`, `--status`, `--result`, `--workload`, `--repository`, `--high-priority`, `--limit` |
| `veeamgo get job history --name "Daily Backup" [--states Running] [--results Failed] [--limit 100]` | Retrieve historical sessions for a job. | `--name`, `--type`, `--states`, `--results`, `--created-since`, `--created-before`, `--ended-since`, `--ended-before`, `--sort`, `--desc`, `--limit` |
| `veeamgo describe job --name "Daily Backup"` | Display live configuration and YAML payload. | `--name` |
| `veeamgo add job --from path/to/job.yaml [--set spec.schedule.retry=3] [--yes] [--dry-run]` | Create jobs from YAML/JSON blueprints or built-in templates. | `--from`, `--template`, `--template-variant`, `--set`, `--dry-run`, `--yes` |
| `veeamgo edit job --name "Daily Backup" [--from updated.yaml] [--from-live] [--set spec.description="Refreshed"] [--yes]` | Update jobs using blueprints, live config exports, or overrides. | `--name`/`--id`, `--from`, `--from-live`, `--template`, `--template-variant`, `--set`, `--dry-run`, `--yes` |
| `veeamgo delete job --name "Old Job" [--yes] [--wait]` | Remove a job. | `--name`/`--id`, `--yes`, `--wait` |
| `veeamgo enable job --name "Daily Backup" [--yes] [--wait]` | Enable a disabled job. | `--name`/`--id`, `--yes`, `--wait` |
| `veeamgo disable job --name "Daily Backup" [--yes] [--wait]` | Disable a job. | `--name`/`--id`, `--yes`, `--wait` |
| `veeamgo start job --name "Daily Backup" [--active-full] [--start-chained]` | Start a job on-demand. | `--name`/`--id`, `--active-full`, `--start-chained`, `--sync-restore-points`, `--yes` |
| `veeamgo stop job --name "Daily Backup" [--graceful=false] [--cancel-chained]` | Stop a running job. | `--name`/`--id`, `--graceful`, `--cancel-chained`, `--yes` |
| `veeamgo retry job --name "Daily Backup" [--start-chained]` | Retry a job that ended with warnings or errors. | `--name`/`--id`, `--start-chained`, `--yes` |
| `veeamgo start job quickbackup --vm-name "VM01" [--name JobName]` | Launch a Quick Backup for a VM (auto-selects eligible jobs). | `--vm-name`, `--name`/`--id`, `--yes` |
| `veeamgo clone job --name "Template Job" [--yes]` | Clone an existing job into a new blueprint-driven job. | `--name`/`--id`, `--yes` |
| `veeamgo template job --list` | List built-in job templates and variants. | None |
| `veeamgo template job --type VSphereBackup --variant full [--write path] [--no-comments]` | Render a job template to stdout or file. | `--type`, `--variant`, `--write`, `--no-comments` |

#### Backups & Storage
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get backup list [--backup *daily*] [--name "Daily Backup"] [--job-id <uuid>] [--limit 50]` | List backups and their repositories. | `--backup`, `--name`, `--job-id`, `--job-type`, `--created-since`, `--created-before`, `--sort`, `--desc`, `--limit` |
| `veeamgo get backup files --name "Daily Backup" [--file *.vbk] [--gfs-period Weekly]` | Show backup files within a job or backup. | `--name`/`--backup`, `--file`, `--gfs-period`, `--created-since`, `--created-before`, `--sort`, `--desc`, `--limit` |
| `veeamgo get backup objects --name "Daily Backup"` | List protected objects (VMs, agents) per backup. | `--name`/`--backup` |

#### Restore & Replica Points
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get restorepoint --name "Daily Backup" [--backup BackupCopy] [--restorepoint VM01*]` | Enumerate restore points for a backup. | `--name`/`--backup`, `--restorepoint`, `--object`, `--platform`, `--malware`, `--created-since`, `--created-before`, `--sort`, `--desc`, `--limit` |
| `veeamgo get replica --name "Replica Job" [--replica vm01] [--replica-point point01]` | List replica restore points for a replication job. | `--name` (required), `--replica`, `--replica-point`, `--platform`, `--malware`, `--created-since`, `--created-before`, `--sort`, `--desc`, `--limit` |

#### Published Disks (Data Integration)
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get publisheddisk --id <mount-id>` | Show details about a specific published disk mount. | `--id` |
| `veeamgo get publisheddisk --all [--limit 20]` | List active published disk mounts. | `--all`, `--limit` |
| `veeamgo start publishdisk --restore-point-id <uuid> --allowed-ip 10.0.0.10 [--disk disk1.vmdk] [--wait]` | Publish disks from a restore point via iSCSI. Requires at least one `--allowed-ip`. | `--restore-point-id`, `--disk`, `--allowed-ip`, `--wait` |
| `veeamgo stop publishdisk --id <mount-id> [--wait]` | End one or more published disk sessions. Repeat `--id` to unpublish several mounts. | `--id`, `--wait` |

### 3.4 Sessions, Tasks & Monitoring
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get session list [--type Backup] [--name *Daily*] [--result Failed]` | List job/session executions with filtering and sorting. | `--name`, `--job`, `--type`, `--state`, `--result`, `--created-since`, `--created-before`, `--ended-since`, `--ended-before`, `--sort`, `--desc`, `--limit` |
| `veeamgo describe session --id <uuid>` | Show metadata for a single session. | `--id` |
| `veeamgo get session logs --id <uuid> [--status Warning]` | Fetch session log records filtered by status. | `--id`, `--status` |
| `veeamgo get task list [--session-id <uuid>] [--type Task]` | List low-level tasks associated with sessions. | `--session-id`, `--type`, `--state`, `--result`, `--limit`, `--sort`, `--desc` |
| `veeamgo describe task --id <uuid>` | Display detailed task information. | `--id` |
| `veeamgo get task logs --id <uuid>` | Retrieve task log entries. | `--id`, `--status` |

### 3.5 Security & Threat Detection
| Command | Purpose | Primary Flags |
| --- | --- | --- |
| `veeamgo get securityanalyzer results` | Show the latest Security & Compliance Analyzer findings. | None |
| `veeamgo describe securityanalyzer schedule` | Display analyzer scheduling and notification configuration. | None |
| `veeamgo start securityanalyzer [--yes] [--wait]` | Launch the analyzer immediately; optionally wait and print results. | `--yes`, `--wait` |
| `veeamgo get malwaredetectionevent [--type YaraScan] [--severity Infected] [--detected-since 2024-01-01T00:00:00Z]` | List suspicious activity events. | `--type`, `--state`, `--source`, `--severity`, `--created-by`, `--engine`, `--machine-name`, `--backup-object`, `--detected-since`, `--detected-before`, `--sort`, `--desc`, `--limit` |
| `veeamgo get yararule` | List YARA rule packages available on the server. | None |

### 3.6 Notes on Reserved Verbs
The `veeamgo migrate` verb is reserved for future workflows and currently has no subcommands.

## 4. Troubleshooting & Tips
- Use `veeamgo get session logs --id <uuid>` (or the task equivalent) to diagnose failed operations.
- Switch to JSON output (`--output json`) when you need exact API fields for scripting.
- If authentication fails, verify the REST URL is HTTPS, confirm certificates, and retry with `--insecure` only in isolated labs.
- Use `veeamgo login --set-default` after onboarding a new profile so subsequent commands reuse it automatically.
- `veeamgo logout --all` is a quick way to ensure cached tokens are removed before handing over a machine.

## 5. Additional Resources
- The project `README.md` includes download instructions, quick-start steps, and release notes.
- Historical design documents live in `develop_docs/` (ignored by git to keep the repository lean).
- Every command exposes contextual help via `veeamgo <...> --help`; prefer that for exhaustive flag descriptions or when new releases introduce additional options.
