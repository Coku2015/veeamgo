# VeeamGo CLI ✨

English | [中文指南](README.zh-CN.md)

VeeamGo is a lightweight CLI for exploring Veeam Backup & Replication (VBR) environments via the official REST API. It gives auditors, operators, and support engineers fast read-only insight without touching the UI. _Crafted through Vibe Coding workflows 🤖🎶._

## Prerequisites
- Access to a Veeam Backup & Replication v13.x server with the REST API enabled (HTTPS port 9419 reachable).
- A VBR account (or OAuth client) with at least read access so `veeamgo login` can authenticate successfully.
- macOS 12+/Linux (glibc 2.31+)/Windows 10+ on x86_64 or arm64. Building from source requires Go 1.25+.
- Outbound network access from your terminal to the VBR REST endpoint you plan to query.

## Download
- Grab the latest signed binaries from the [GitHub Releases page](https://github.com/Coku2015/veeamgo/releases/latest).
- Each release ships platform-specific binaries for macOS, Linux, and Windows; download the asset that matches your OS/arch and rename it to `veeamgo` (or `veeamgo.exe`).
- Optional: add the folder to your `PATH` or copy the binary to a directory already on the `PATH`.

```bash
curl -fSL "https://github.com/Coku2015/veeamgo/releases/download/v13.0.1/veeamgo_linux_amd64" -o veeamgo && chmod +x veeamgo
```

## Quick Start
1. Download the binary that matches your platform and rename it to `veeamgo` (or `veeamgo.exe`).
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
