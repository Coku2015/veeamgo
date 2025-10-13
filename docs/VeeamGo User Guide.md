# VeeamGo User Guide

## Overview

`veeamgo` is a read‑only command line client for the Veeam Backup & Replication (VBR) REST API. The CLI is organised around a verb‑first grammar:

```
veeamgo <verb> <resource> [subcommand] [flags]
```

The primary verbs are:

- `get` – list or retrieve collections of resources.
- `describe` – show a single resource in detail.
- `rescan` – trigger inventory rescans.

Authentication and profile management are handled with the dedicated `login` / `logout` commands.

All tabular timestamps are rendered in the VBR server’s time zone using the format `YYYY-MM-DD HH:MM:SS (+HH:MM)`.

## Global Options

All commands support the following global flags:

| Flag | Description |
| ---- | ----------- |
| `--config <path>` | Override the configuration file location (defaults to `~/.veeamgo/config.yaml`). |
| `--profile <name>` | Explicitly select a stored profile; defaults to the profile set during login. |
| `--output table|json` | Choose between human-readable tables (default) and raw JSON passthrough. |

## Authentication & Profiles

### Log in to VBR

```
veeamgo login \
  --server https://vbr.example.com:9419 \
  --username ADMIN \
  [--password *****] \
  [--insecure] \
  [--set-default] \
  [--save-password]
```

- Prompts for the password if `--password` is omitted.
- Stores the session token in the encrypted session cache; optionally persists the password for silent refresh.

### Log out

```
veeamgo logout [--all]
```

- Removes the cached session for the active profile.
- Use `--all` to clear every cached profile session.

## Getting Data (`veeamgo get`)

### Server

```
veeamgo get server info
veeamgo get server time
veeamgo get server managed [--type <Type>] [--name <pattern>] [--limit N]
```

- `info` – returns general server metadata.
- `time` – reports the server clock in table and JSON formats.
- `managed` – lists managed servers. Filter by type (`WindowsHost`, `LinuxHost`, `HyperVHost`, …), name wildcard, or limit.

### Repository

```
veeamgo get repository [--type <Type>] [--name <pattern>] [--limit N]
```

Lists backup repositories with capacity statistics. Use `--type` (e.g., `LinuxLocal`, `WinLocal`, `SOBR`) or name wildcards.

### Job

```
veeamgo get job \
  [--name <pattern>] \
  [--type <Type> ...] \
  [--vsphere-backup] [--hyperv-backup] [--vsphere-replica] \
  [--cloud-director-backup] [--entra-tenant-backup] [--entra-auditlog-backup] \
  [--file-backup-copy] [--legacy-backup-copy] [--backup-copy] \
  [--windows-agent-backup] [--linux-agent-backup] [--entra-tenant-backup-copy] \
  [--status <Status>] [--result <Result>] \
  [--workload <Platform>] [--repository <Repository Name>] \
  [--high-priority] \
  [--since <RFC3339>] [--before <RFC3339>] \
  [--after-job <Name>] \
  [--sort <column>] [--desc] \
  [--limit N]
```

Use the type switches to quickly narrow by workload. Sorting supports columns such as `name`, `lastRun`, or `nextRun`.

### Restore Points

```
veeamgo get restorepoint \
  --name "<Job Name>" \
  [--backup "<Backup Name>"] \
  [--restorepoint "<Pattern>"] \
  [--object <Object ID>] \
  [--platform <Platform Name>] \
  [--platform-id <Platform ID>] \
  [--malware <Status>] \
  [--created-since <RFC3339>] \
  [--created-before <RFC3339>] \
  [--sort <column>] [--desc] \
  [--limit N]
```

`--name` identifies the backup job. If a job has multiple backups, provide `--backup` to pick one explicitly.

### Replica Restore Points

```
veeamgo get replica \
  --name "<Replication Job>" \
  [--replica "<Replica Name>"] \
  [--replica-point "<Pattern>"] \
  [--platform <Platform Name>] \
  [--platform-id <Platform ID>] \
  [--malware <Status>] \
  [--created-since <RFC3339>] \
  [--created-before <RFC3339>] \
  [--sort <column>] [--desc] \
  [--limit N]
```

When a job protects multiple replicas, add `--replica` to disambiguate.

### Proxies

```
veeamgo get proxy \
  [--name <pattern>] \
  [--type <Type>] \
  [--host-id <HostID>] \
  [--sort <column>] [--desc] \
  [--limit N]
```

Types include values such as `ViProxy`, `HvProxy`, `GeneralPurposeProxy`. Table output already normalises these to friendly labels.

### Traffic Rules

```
veeamgo get trafficrule
```

Shows preferred network and throttling rules, including encryption and bandwidth limits.

### Exclusion VMs

```
veeamgo get exclusionvm [--sort <column>] [--desc] [--limit N]
```

Lists globally excluded VMs (name, platform, notes). Use `describe` (see below) with the exclusion ID for details.

### Backups

#### Files

```
veeamgo get backup files \
  --name "<Job Name>" \
  [--backup "<Backup Name>"] \
  [--file "<pattern>"] \
  [--gfs <GFS Period>] \
  [--created-since <RFC3339>] \
  [--created-before <RFC3339>] \
  [--sort <column>] [--desc] \
  [--limit N]
```

Reports backup file sizes, dedupe/compression ratios, creation times, and GFS tiers. GFS headings are blank when no tier applies.

#### Objects

```
veeamgo get backup objects \
  --name "<Job Name>" \
  [--backup "<Backup Name>"]
```

Lists protected objects (name, type, platform, restore point count). The standalone alias `veeamgo get objectsinbackup` accepts the same options.

### Sessions

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

Use the nested command to inspect log records for a single session:

```
veeamgo get session logs <session-id> [--status <Status>]
```

### License

```
veeamgo get license
veeamgo get license sockets [--name <pattern>] [--host-name <pattern>] [--host-id <ID>] [--type <Type>] [--sort <column>] [--desc] [--limit N]
veeamgo get license instances [--name <pattern>] [--host-name <pattern>] [--type <Type>] [--instance <ID>] [--sort <column>] [--desc] [--limit N]
veeamgo get license capacity [--name <pattern>] [--type <Type>] [--limit N]
```

- `license` (or `license summary`) – summary of the current license state.
- `license sockets` / `license instances` / `license capacity` – workload usage breakdowns with extensive filtering.

### Security Analyzer

```
veeamgo get security analyzer send-results
```

Returns the latest send-results best practice assessment.

## Describing Resources (`veeamgo describe`)

### Repository

```
veeamgo describe repository --name "<Repository Name>"
```

Displays repository metadata, including type, capacity, online state, and the raw configuration rendered as YAML.

### Proxy

```
veeamgo describe proxy --name "<Proxy Name>" [--type <Type>]
```

Shows detailed proxy configuration. Add `--type` if multiple proxies share the same name across workloads.

### Job

```
veeamgo describe job "<Job Name>"
```

Returns job definition, scheduling, repository, and last session identifiers.

### Session

```
veeamgo describe session <session-id>
veeamgo describe session current
```

- `current` – prints cached authentication profile information.
- With a specific session ID, returns job/session metadata and embedded YAML details.

### Configuration Backup

```
veeamgo describe configurationbackup
```

Summarises the configuration backup policy (repository, GFS retention, encryption, last run).

### Exclusion VM

```
veeamgo describe exclusionvm <exclusion-id>
```

Use the ID from `veeamgo get exclusionvm` to inspect a specific exclusion (inventory references and notes).

### Security Analyzer Schedule

```
veeamgo describe security analyzer schedule
```

Shows the daily scan configuration, notification recipients, and custom notification settings.

## Rescanning Resources (`veeamgo rescan`)

### Repositories

```
veeamgo rescan repository [--id <RepositoryID> ... | --all] [--wait]
```

Kick off repository rescans. Supply one or more `--id` values (run `get repository` to find them) or use `--all`. Add `--wait` to poll the resulting session until completion.

### Managed Servers

```
veeamgo rescan server [--id <ServerID> | --all] [--wait]
```

Requests a rescan for a specific managed server or every managed server on the VBR instance.

## Security & Traffic Utilities

- `veeamgo get trafficrule` – view preferred networks and bandwidth limits.
- `veeamgo get exclusionvm` / `describe exclusionvm` – manage global VM exclusion visibility.
- `veeamgo get security analyzer send-results` / `describe security analyzer schedule` – review Security & Compliance Analyzer configuration and results.

## Shell Completion

Generate shell completion scripts with:

```
veeamgo completion <bash|zsh|fish|powershell>
```

Follow the shell-specific instructions printed by Cobra to install the completion script.

## Working With JSON Output

`--output json` prints the raw payload returned by VBR. Use this when chaining `jq`, persisting data, or debugging. The tabular view is intended for interactive exploration and uses the friendly time format described earlier.

## Typical Workflow

1. Authenticate once: `veeamgo login --server … --username …`.
2. Explore resources with `veeamgo get …`.
3. Pivot to detailed views via `veeamgo describe …`.
4. Trigger repository/server rescans when inventory needs refreshing.
5. When finished, optionally run `veeamgo logout` to clear cached credentials.

For further automation, combine the JSON output mode with existing scripts (see `scripts/veeamgo_e2e.sh` for end‑to‑end examples).
