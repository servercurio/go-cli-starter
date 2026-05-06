# CLAUDE.md

Guidance for Claude Code working in this repository. This file covers project intent, conventions, and procedural guidance that isn't captured in the agent reference docs:

- [`.claude/instructions.md`](.claude/instructions.md) — tech stack, personality, requirements
- [`.claude/build-commands.md`](.claude/build-commands.md) — `task` targets and what they run
- [`.claude/module-structure.md`](.claude/module-structure.md) — directory-by-directory roles
- [`.claude/conventions.md`](.claude/conventions.md) — coding conventions for this repo
- [`.claude/git-hooks.md`](.claude/git-hooks.md) — required local git hooks (must be installed per clone)

## What this project is

A **starter template**, not an application. It provides HTTP server scaffolding (Echo v5, TLS, logging, config) and intentionally omits persistence, auth, validation, and business logic — those are decisions for downstream consumers.

When asked to add functionality, prefer keeping the codebase generic and composable. Don't introduce app-specific concerns (a particular database driver, an auth scheme, a domain model) unless explicitly requested.

## Local environment vs binary defaults

The `Taskfile.yaml` env block sets `APP_DAEMON_LOG_LEVEL=trace`, enables HTTPS, and uses ports 8888/4443 — those are developer-convenience overrides, **not** the binary's defaults. The binary itself defaults to HTTP on `:8080` with HTTPS disabled. Don't assume Taskfile env values when reasoning about production behavior.

A fresh clone won't have `vendor/` until `task vendor` (or any task that depends on it, like `task build`) runs.

## When adding code

- **New route**: add a file under `internal/api/v1/` returning a `router.Route`, then register it via `module.WithRoutes(...)` in `internal/api/v1/module.go`. Don't bypass the module abstraction by calling `e.GET(...)` directly on the Echo instance. After adding/changing routes, run `task openapi:gen` to refresh `docs/openapi.yaml` — CI's OpenAPI Drift check fails the PR if the checked-in spec doesn't match what the code now produces.
- **New top-level module**: register it in `cmd/daemon/main.go` alongside the `api` module.
- **New config field**: add to the appropriate struct under `internal/application/config_*.go`, give it a sensible default in `DefaultConfig()`, wire env-var loading using the helpers in `internal/env/`, and extend the struct's `Validate() error` to reject obviously-bad values — the daemon refuses to boot when `Application.Configure` returns a non-nil error, so validation lives there rather than at request time. When adding a brand-new config struct, also implement `Validate()` and aggregate it into `Config.Validate()` (`internal/application/config.go`).
- **New middleware**: if globally applied, add to the middleware stack in `internal/application/application.go`. If module-scoped, pass via `module.WithMiddleware(...)`.
- **New CI workflow**: file under `.github/workflows/` following the naming convention in `.github/workflows/docs/naming-standards.md` (`ddd-xxxx-name.yaml` file, matching `ddd: [XXXX] Name` workflow `name:`). PR-triggered workflows use the **200** prefix in this repo (the upstream-standard CITR slot, repurposed locally); main-branch push-triggered workflows use **300**; operational/release flows use **100**; reusable workflows use **800**. New `go:generate` directives don't need a workflow change — `task generate` is invoked by both PR reusable workflows and the release flow, and picks them up via `./...`.
- **New SQL migration**: drop a `YYYYMMDDHHMMSS_description.sql` file (Goose convention with `-- +goose Up` / `-- +goose Down` markers) into `internal/database/migrations/sql/`. The `//go:embed` pattern picks it up at compile time — no Go code change needed. `database.Migrate` runs the up direction on every successful daemon boot.
- **New domain model**: add a Bun struct type in a new file under `internal/database/orm/`; access the connection via `orm.Database()`. Don't write directly to `database.Connection()` from app code — go through the ORM so dialect changes stay isolated.
- **New health component**: register a `health.CheckFunc` against `app.HealthRegistry()` from inside `Application.registerHealthChecks` (preferred — keeps lifecycle ownership of registration), or from `cmd/daemon/main.go` if the dependency lives outside the `internal/application/` package. Closures run on every `/readyz` request, so keep them sub-second; put expensive probes behind a cached background poller. Conditionally register subsystems that may be disabled — only active components should appear in the report.
- **Richer OpenAPI metadata**: declare endpoint metadata via the builder options on `internal/api/std/endpoint` — `WithSummary`, `WithDescription`, `WithRequest(value)`, `WithResponse(code, value, "description")`. Pass a zero value of the request/response Go type (e.g. `WithResponse(200, health.Report{}, "OK")`); the reflection-based schema generator at `internal/openapi/schema.go` produces the JSON Schema and registers it under `components.schemas` with a `$ref` from the operation. Don't reach for comment annotations or external generators — the builder-options approach matches the rest of the project's idiom and keeps everything in one place.
- **New schema-relevant Go construct**: extend `internal/openapi/schema.go` (`schemaFor` for new `reflect.Kind`s, `inlineStructSchema` for new struct-tag handling). Keep the inline-vs-$ref split: named struct types ⇒ `$ref`, primitives/slices/maps/anon structs ⇒ inline. Add a fixture-based test in `internal/openapi/schema_test.go` that pins the expected JSON Schema shape so future refactors can't silently change the contract.
- **Release behaviour change**: edit `.releaserc.json` for semantic-release plugin config (commit-analyzer rules, release notes preset, exec hooks, branch channels, GitHub asset list). Cross-check that any new binary or other artifact added to `bin/` is also listed under `@semantic-release/github` `assets` so it ships with the release, AND that the `subject-path` glob in `.github/workflows/800-call-semantic-release.yaml`'s `Attest provenance for file artifacts` step includes it — every release artifact must carry a GitHub-signed Sigstore attestation. Add an `actions/attest-sbom` step too if the artifact has a corresponding SBOM.
- **New Helm chart change**: edit files under `charts/go-echo-starter/templates/` (or add a new template). Run `task helm:lint` and `task helm:template` after every change — `helm:template` overlays TLS and the monitoring API versions so gated resources are validated. New optional resources (ServiceMonitor / PodMonitor / PodLogs etc.) MUST be gated by an `.enabled` value defaulting to `false` and (for CRD-dependent resources) wrapped in `{{- if .Capabilities.APIVersions.Has "<group>/<version>" -}}` so the chart installs cleanly on clusters without the CRD. CI's `Helm Lint` and `Helm Install` jobs (`800-call-helm-lint.yaml` / `800-call-helm-install.yaml`) fail the PR if the chart is invalid or fails to install on a kind cluster. New chart values that affect the rendered manifest must also be exercised by `charts/go-echo-starter/ci/full-values.yaml`.

## Supply-chain signing policy

This project signs **hashes, not artifacts**. `task sign` produces only `<binary>.sha256.asc` (never `<binary>.asc`); `task helm:sign` produces only `<chart>.tgz.sha256.asc`. SBOMs are themselves manifests and ARE signed directly (`sbom.json.asc`, `container-sbom.json.asc`, `helm-sbom.json.asc`). The signed hash transitively pins the artifact. Don't add bare `<file>.asc` entries (where `<file>` is a binary or `.tgz`) to the release asset list or to any new signing target. In addition, every release artifact and every OCI subject (container image, helm OCI artifact) is attested via `actions/attest-build-provenance` and (for SBOMs) `actions/attest-sbom`. Verification path going forward: `gh attestation verify <artifact> --owner servercurio` for primary verification, GPG-on-hashes for offline-capable secondary verification.

## Things to leave alone unless asked

- This repo is on Echo v5. Don't revert v5 idioms or pin older versions.
- `obfusicate/` exposes one tiny helper (`ConcealPrefix`) used for log redaction — leave it alone unless the task is explicitly about redaction; the misspelled package name is intentional and matches the import path.
