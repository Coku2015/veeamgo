# VeeamGo CLI

VeeamGo is a command line client for exploring Veeam Backup & Replication (VBR) environments over the official REST API. It is designed for auditors, support engineers, and operations teams that need fast visibility into VBR metadata without touching the UI or modifying state.

Key features:

- Verb-first command grammar (`veeamgo get …`, `veeamgo describe …`, `veeamgo rescan …`) .
- Human-friendly tables with consistent timestamp formatting and automatic unit conversions, plus optional raw JSON passthrough for automation.
- Authentication profile management (`login` / `logout`) with encrypted session caching and optional password persistence.
- Coverage for the read-only inventory surface: jobs, repositories, proxies, restore points, replica points, license usage, security analyzer, traffic rules, and more.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Building](#building)
3. [Configuration & Authentication](#configuration--authentication)
4. [Usage](#usage)
   - [Global Structure](#global-structure)
   - [Common Commands](#common-commands)
5. [JSON Output](#json-output)
6. [Development](#development)
7. [Documentation](#documentation)
8. [License](#license)

## Download




## Prerequisites

- Go 1.19 or higher.
- Access to a VBR server with the REST API enabled (default TCP port 9419).
- Optional: `jq` for JSON post-processing.

## Building

Clone the repository and run:

```bash
go build -o veeamgo ./cmd
```

For cross-platform release artefacts we disable CGO and reuse local build caches:

```bash
CGO_ENABLED=0 GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache \
  GOOS=darwin  GOARCH=amd64 go build -o dist/veeamgo-darwin-amd64 .
# Repeat for other GOOS/GOARCH combinations as needed
```

## Configuration & Authentication

Run `veeamgo login` to authenticate against the VBR REST API. A configuration file is stored under `~/.veeamgo/config.yaml`, and session tokens are encrypted in `~/.veeamgo/sessions.json`.

```bash
veeamgo login \
  --server https://vbr.example.com:9419 \
  --username administrator \
  --save-password \
  --set-default
```

Subsequent commands reuse the cached session. Clear it with:

```bash
veeamgo logout        # Clears the active profile
veeamgo logout --all  # Clears every cached session
```

## Usage

### Global Structure

All resource commands follow a verb-first grammar:

```
veeamgo <verb> <resource> [subcommand] [flags]
```

Primary verbs:

- `get` – list resources.
- `describe` – show a single resource in detail.
- `rescan` – initiate inventory rescans.

Global flags available everywhere:

| Flag | Description |
| ---- | ----------- |
| `--config <path>` | Override config file path (default `~/.veeamgo/config.yaml`). |
| `--profile <name>` | Select a stored profile; defaults to the active profile. |
| `--output table|json` | Choose table output or raw JSON. |

Timestamps in table output respect the VBR server’s timezone and use the format `YYYY-MM-DD HH:MM:SS (+HH:MM)`.

### Common Commands

Fetch high-level information:

```bash
veeamgo get server info
veeamgo get license
veeamgo get job --limit 20
```

Inspect specific resources:

```bash
veeamgo describe repository --name "Default Backup Repository"
veeamgo describe proxy --name "Proxy-01"
veeamgo describe security analyzer schedule
```

List backups, restore points, and replicas using the job’s name:

```bash
veeamgo get backup files --name "Backup Job 1" --limit 10 --output table
veeamgo get restorepoint --name "Backup Job 1" --limit 20 --output json
veeamgo get replica --name "Replication Job" --limit 10
```

View global configuration and rules:

```bash
veeamgo get trafficrule
veeamgo get exclusionvm
veeamgo get security analyzer send-results
```

Kick off rescans:

```bash
veeamgo rescan repository --all --wait
veeamgo rescan server --id abcd-1234 --wait
```

For a full catalogue of commands, flags, and examples consult the [User Guide](docs/user-guide.md).

## JSON Output

Supply `--output json` to view the exact API payload (including pagination blocks where applicable). This is ideal when passing data to `jq`, recording audit snapshots, or integrating with other tooling. The table output is intended for interactive use and already applies friendly column names and formatting.

## Development

- Run unit tests with `go test ./...`.
- Linting/formatting follows standard Go tooling (`gofmt`, `goimports`, `golangci-lint`).
- The end-to-end smoke script (`scripts/veeamgo_e2e.sh`) exercises every read-only command; update it alongside behavioural changes.
- When adding new time-based fields, format them through the shared helpers (`formatTimestamp`, `formatTimestampValue`, `formatAPITime`) to honour the server timezone and display format.

## Documentation

- [User Guide](docs/user-guide.md) – comprehensive CLI reference.
- [Stage 3B UI Adjustments (archived)](docs/archived-stage-3b-ui-adjustments.md) – summary of the UI work delivered in Stage 3B.
- Additional project notes and requirements live under the `docs/` directory.

## License

This project is provided under the MIT License. See [LICENSE](LICENSE) for details.
