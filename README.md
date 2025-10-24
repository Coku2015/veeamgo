# VeeamGo CLI

English | [中文指南](README.zh-CN.md)

VeeamGo is a lightweight CLI for exploring Veeam Backup & Replication (VBR) environments via the official REST API. It gives auditors, operators, and support engineers fast read-only insight without touching the UI.

## Download
- Grab the latest signed binaries from the [GitHub Releases page](https://github.com/Coku2015/veeamgo/releases/latest).
- Each release ships archives for macOS, Linux, and Windows; extract the archive and keep the `veeamgo` (or `veeamgo.exe`) binary.
- Optional: add the folder to your `PATH` or copy the binary to a directory already on the `PATH`.

```bash
# Example: macOS / Linux
curl -LO https://github.com/Coku2015/veeamgo/releases/latest/download/veeamgo-darwin-arm64.tar.gz
tar -xzf veeamgo-darwin-arm64.tar.gz
chmod +x veeamgo
```

## Quick Start
1. Download and unpack the binary that matches your platform.
2. (Optional) Move it to `/usr/local/bin/` (macOS/Linux) or a folder listed in `%PATH%` (Windows).
3. Sign in: `veeamgo login --server vbr.example.com --username administrator --insecure`.
4. Run a command, e.g. `veeamgo get job --limit 5` for a quick inventory check.
5. Sign out when done: `veeamgo logout` (or `veeamgo logout --all` to clear every cached session).

## Authentication Essentials
- The first `login` prompts for a password (or reads `VEEAMGO_PASSWORD`) and stores the profile under `~/.veeamgo/config.yaml`.
- Sessions are cached securely; subsequent commands reuse the active profile until you call `logout`.
- Add `--save-password` to reuse credentials after reboots, or `--set-default` to make the profile the default target.
- Use `veeamgo logout --all` to clear every cached session before switching labs or sharing the machine.

## Build from Source (Optional)
If you prefer a local build or need to debug upcoming changes:

```bash
go build -o veeamgo ./cmd
```

Inside sandboxed environments set `GOCACHE=$(pwd)/.gocache` (and `GOMODCACHE=$(pwd)/.gomodcache`) to reuse local Go caches.

## Where To Go Next
- Read the full command reference in `docs/UserGuide.md`.

