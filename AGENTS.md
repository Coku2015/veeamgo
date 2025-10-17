# Repository Guidelines

## Project Structure & Module Organization
The CLI entrypoints live under `cmd/`, grouped by verb (for example, `cmd/scaleout_repository.go`). Non-exported business logic sits in `internal/` (API clients, auth, config helpers), while reusable utilities reside in `pkg/`. Documentation, OpenAPI specs, and user guides are stored in `docs/`; examples and fixtures are colocated there. Automation scripts belong in `scripts/`, and release binaries are generated into `dist/`.

## Build, Test, and Development Commands
- `go build -o veeamgo ./cmd` – compile the CLI with local module cache reuse (`GOCACHE=$(pwd)/.gocache` when inside the sandbox).  
- `go test ./...` – run unit tests for all packages.  
- `go test -tags=integration ./integration/...` – execute integration suites against a lab VBR server (requires env credentials).  
- `golangci-lint run ./...` – enforce formatting, vetting, and static checks.  
- `spectral lint docs/swagger/openapi.json` – validate the split OpenAPI definition.
- Always prefer project-local caches when invoking Go tooling. Set `GOCACHE=$(pwd)/.gocache` (and `GOMODCACHE=$(pwd)/.gomodcache` if needed) instead of relying on platform defaults such as `~/Library/Caches/go-build`, which are typically inaccessible in the sandbox.

## Coding Style & Naming Conventions
All Go code must be formatted with `gofmt`/`goimports`; run via `golangci-lint`. Exported identifiers use PascalCase, package names are lower_snake, and errors wrap underlying causes with `fmt.Errorf("context: %w", err)`. CLI commands follow `veeamgo <verb> <resource>` with flags favouring human-readable keys (`--name`, `--limit`). Timestamps printed in tables must go through `formatTimestamp`, `formatTimestampValue`, or `formatAPITime`.

## Testing Guidelines
Unit tests are table-driven and colocated (`*_test.go`). Achieve ≥80% coverage on new packages and include edge/error cases. Integration tests sit behind the `integration` build tag and consume environment variables such as `VEEAMGO_SERVER`. Update `scripts/veeamgo_e2e.sh` alongside new commands to maintain smoke coverage.

## Commit & Pull Request Guidelines
Use conventional commits where practical (e.g., `feat(job): add job get command`). Every pull request should link related issues, summarise behavioural changes, and note evidence (tests, screenshots, sample CLI output). Before requesting review ensure `go test ./...`, `golangci-lint run ./...`, and documentation updates succeed.

## Security & Configuration Tips
Never commit credentials; rely on environment variables (`VEEAMGO_SERVER`, `VEEAMGO_USERNAME`, `VEEAMGO_PASSWORD`) or encrypted profiles stored under `~/.veeamgo`. HTTPS is mandatory—`--insecure` is reserved for lab usage. When credentials exist in the profile, the CLI auto-refreshes tokens, so contributors must keep config handling deterministic and free of plaintext secrets.
