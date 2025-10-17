# VeeamGo Development Guidelines

## 1. Core Principles
- **API fidelity**: Client models, fields, enums, and request shapes must align with the Veeam REST API v1.3 specification. Do not introduce attributes that are absent from the official schema and avoid endpoints marked as deprecated.
- **Respect OpenAPI semantics**: Treat summaries that start with “Get …” as read-only flows even if the underlying HTTP verb is POST/PUT. Surface these operations through the `get`/`describe` verbs in the CLI.
- **English-only artifacts**: Source code, comments, docs, help text, log messages, and sample outputs must be written in English.
- **Security first**: Enforce HTTPS by default, store secrets in the OS keychain or encrypted cache, and redact sensitive data in every output path.
- **Consistency**: Commands follow the `veeamgo <verb> <resource>` pattern (for example, `veeamgo get repository`) with uniform flag ergonomics across the CLI.
- **Automation friendly**: Every command should support machine-readable output (`--output json`) and deterministic exit codes.
- **Raw JSON passthrough**: When `--output json` is selected, commands must emit the exact API payload (maps/slices or captured `detail.Raw`) without reshaping or renaming fields.

## 2. Development Workflow
1. **Research**  
   Review the relevant portion of `docs/swagger/openapi.json` and consult the official API docs for behavioural nuances.
2. **Design**  
   Define request/response structs inside `internal/models`, plan command UX (flags, examples) with kubectl-style alignment, and validate name/ID resolution requirements.
3. **Implementation**  
   Wire command factories under `cmd/`, implement API calls within `internal/client`, and push reusable helpers into `pkg/` when appropriate.
4. **Testing**  
   Add table-driven unit tests for new packages and extend integration tests (behind the `integration` build tag) when the feature interacts with live VBR resources.
5. **Documentation**  
   Update the user guide, UI samples, changelog, and any runbooks affected by the change.

## 3. Error Handling & Logging
- Return user-friendly errors that include actionable remediation tips.
- When the REST API returns structured error payloads (`message`, `title`, `errorCode`), parse them and surface a natural-language message instead of dumping raw JSON.
- Wrap errors with `fmt.Errorf("context: %w", err)` for traceability.
- Logging levels: `error` (unexpected failures), `warn` (recoverable issues), `info` (high-level flow), `debug` (request metadata/timing), `trace` (request/response bodies with tokens redacted).
- Structured logs should include `component`, `operation`, `resource`, and `duration` fields.

## 4. Configuration Management
- Default config search order: `$HOME/.config/veeamgo/config.yaml`, legacy `$HOME/.veeamgo.config`, and any path supplied through `VEEAMGO_CONFIG` or `--config`.
- Environment overrides use the `VEEAMGO_` prefix (for example, `VEEAMGO_SERVER`, `VEEAMGO_PASSWORD` / `VEEAMGO_PASS`).
- Support multiple named profiles (`veeamgo config profile create|list|use|delete`) and persist sessions in secure storage (OS keychain or encrypted file).
- Provide a ready-to-clone template at `docs/examples/veeamgo.config.example.yaml`; customers can copy it and adjust values.
- When credentials are present in the profile, commands should auto-authenticate or refresh tokens so that manual `veeamgo login` remains optional.

## 5. Coding Standards
- Format code with `gofmt`/`goimports` and run `golangci-lint` before committing.
- Avoid global state; prefer dependency injection via constructors and pass `context.Context` through every API boundary.
- Document exported functions with GoDoc comments and keep command functions focused on orchestration.
- Reuse shared helpers (for example, `cmd/names.go`, output utilities) instead of duplicating logic.
- Format timestamps through `formatTimestamp`, `formatTimestampValue`, or `formatAPITime` so table output reflects the VBR server time zone (`YYYY-MM-DD HH:MM:SS ±HH:MM`).

## 6. Command UX & Parameter Conventions
- Commands follow `<verb> <resource>` immediately after the `veeamgo` root and should reference shared constants in `cmd/names.go` when renaming.
- Prefer human-friendly identifiers (`--name`, wildcardable filters) over raw IDs. Internally resolve names to IDs and emit clear errors when the name is missing, duplicated, or unknown.
- Use `requireFlag` for validation and keep error messages consistent with the `flag --xxx is required (…​)` pattern.
- Maintain parity between help text, docs, and examples—every command requires at least one table sample and one JSON sample in the documentation.
- Honour OpenAPI summary semantics for read-only flows and keep flag naming aligned across subcommands (for example, `--limit`, `--sort`, `--desc`).

## 7. Testing Expectations
- **Unit tests** cover edge cases, error paths, and success paths for new code.
- **Integration tests** (build tag `integration`) require lab credentials and should exercise live API behaviour.
- **E2E smoke script**: keep `scripts/veeamgo_e2e.sh` in sync with new or modified commands. The script is treated as part of the acceptance criteria and must log deterministically for regression analysis.
- **Mocks/Fakes**: generate interfaces with `mockgen` or use in-memory fakes to isolate API client behaviour.
- **CI checks** always include `go test ./...`, `golangci-lint run ./...`, and OpenAPI validation.
- **Tool cache hygiene**: when running Go commands in sandboxes/CI, always point caches to the repository (for example, `GOCACHE=$(pwd)/.gocache`, `GOMODCACHE=$(pwd)/.gomodcache`). Avoid system-level defaults such as `~/Library/Caches/go-build`, which may be unavailable.

## 8. Version Control & Branching
- `main` is the protected, releasable branch.
- Working branch prefixes: `feature/<topic>`, `fix/<bug-id>`, `docs/<topic>`, etc.
- Require passing CI, code review, and updated documentation before merging.
- Use conventional commits when practical (for example, `feat(job): add job get command`).

## 9. Release Process
- Update the CLI version in `cmd/root.go`, refresh documentation, and draft release notes that highlight new commands, flags, and breaking changes.
- Build cross-platform binaries into `dist/`:

  ```bash
  mkdir -p dist
  GOCACHE=$(pwd)/.gocache GOMODCACHE=$(pwd)/.gomodcache \
    GOOS=darwin  GOARCH=amd64  go build -o dist/veeamgo_darwin_amd64 .
  GOCACHE=$(pwd)/.gocache GOMODCACHE=$(pwd)/.gomodcache \
    GOOS=darwin  GOARCH=arm64  go build -o dist/veeamgo_darwin_arm64 .
  GOCACHE=$(pwd)/.gocache GOMODCACHE=$(pwd)/.gomodcache \
    GOOS=linux   GOARCH=amd64  go build -o dist/veeamgo_linux_amd64 .
  GOCACHE=$(pwd)/.gocache GOMODCACHE=$(pwd)/.gomodcache \
    GOOS=linux   GOARCH=arm64  go build -o dist/veeamgo_linux_arm64 .
  GOCACHE=$(pwd)/.gocache GOMODCACHE=$(pwd)/.gomodcache \
    GOOS=windows GOARCH=amd64  go build -o dist/veeamgo_windows_amd64.exe .
  ```

- Publish checksums (and container images if applicable), tag the release (`vX.Y.Z`), and attach artifacts.

## 10. Security Practices
- Never hard-code credentials or tokens and validate TLS certificates unless the user explicitly opts in with `--insecure`.
- Run `gosec ./...` periodically to catch common issues.
- Zero sensitive data in panic logs or stack traces and provide a security contact for vulnerability disclosures.

## 11. Documentation Standards
- Keep user-facing docs in `docs/` synchronized with command behaviour. Update UI samples, troubleshooting steps, and examples whenever commands change.
- Provide table and JSON examples for new commands along with error scenarios.
- Record changelog entries for every new command or breaking flag change to preserve upgrade notes.
