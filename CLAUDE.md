# CLAUDE.md

Guidance for Claude Code working in this repository. This file covers project intent, conventions, and procedural guidance that isn't captured in the agent reference docs:

- [`.claude/instructions.md`](.claude/instructions.md) — tech stack, personality, requirements
- [`.claude/build-commands.md`](.claude/build-commands.md) — `task` targets and what they run
- [`.claude/module-structure.md`](.claude/module-structure.md) — directory-by-directory roles
- [`.claude/conventions.md`](.claude/conventions.md) — coding conventions for this repo
- [`.claude/git-hooks.md`](.claude/git-hooks.md) — required local git hooks (must be installed per clone)

<!-- Auto-load the reference docs above so Claude has them in context from session start. -->
@.claude/instructions.md
@.claude/build-commands.md
@.claude/module-structure.md
@.claude/conventions.md
@.claude/git-hooks.md

## What this project is

A **starter template** for command-line tools and CLI daemons, not an application. It provides Cobra-driven subcommand scaffolding, structured logging, layered configuration, an opt-in PostgreSQL/Bun ORM, and a shared goroutine pool — and intentionally omits domain-specific business logic.

When asked to add functionality, prefer keeping the codebase generic and composable. Don't introduce app-specific concerns (a particular database driver beyond pgx, a specific business model, an auth scheme) unless explicitly requested.

## Local environment vs binary defaults

The `Taskfile.yaml` env block sets `APP_DAEMON_LOG_LEVEL=trace` for developer convenience. The compiled binary's own default is `info`. Don't assume Taskfile env values when reasoning about production behavior.

A fresh clone won't have `vendor/` until `task vendor` (or any task that depends on it, like `task build`) runs.

## When adding code

- **New subcommand**: drop a file under `internal/cli/` returning a `*cobra.Command`. Wire any subcommand-specific flags to a local struct. Register it in `NewRootCommand` (in `internal/cli/root.go`). Inside `RunE`, reach for the already-configured `*Application` via `rc.app`; call `app.Initialize()` then `app.Run(ctx, body)` (one-shot) or `app.RunUntilSignal(ctx, body)` (daemon).
- **New global (persistent) flag**: add it to the `PersistentFlags()` block in `NewRootCommand`, append the field to `rootFlags`, and overlay it onto `*application.Config` inside `prepare()` — gate the overlay on `cmd.Flags().Changed(...)` so unset flag defaults don't clobber values loaded from file/env.
- **New `Application` subsystem**: create a new `application_<name>.go` with `configure<Name>` / `initialize<Name>` / `shutdown<Name>` methods as appropriate, then call them from the lifecycle phases in `application.go`. Don't grow `application.go` itself. The `pool` subsystem (`application_pool.go`) is the canonical reference.
- **New pool-backed worker**: call `app.Pool().SubmitWithContext(ctx, ...)`. Don't construct a one-off `ants.Pool` — using the shared pool means a single `--workers` / `APP_POOL_SIZE` knob bounds total in-flight goroutines.
- **New config field**: add to the appropriate struct (`internal/application/config.go`, `internal/logging/config.go`, `internal/database/config.go`, `internal/pool/config.go`), give it a sensible default in the package's `DefaultConfig()`, wire env-var loading using the helpers in `internal/env/`, and extend the struct's `Validate() error` to reject obviously-bad values — `Application.Configure` returns the joined validation error and the subcommand refuses to run.
- **New CI workflow**: file under `.github/workflows/` following the naming convention in `.github/workflows/docs/naming-standards.md` (`ddd-xxxx-name.yaml` file, matching `ddd: [XXXX] Name` workflow `name:`). PR-triggered workflows use the **200** prefix; main-branch push-triggered workflows use **300**; operational/release flows use **100**; reusable workflows use **800**. New `go:generate` directives don't need a workflow change — `task generate` is invoked by both PR reusable workflows and the release flow, and picks them up via `./...`.
- **New SQL migration**: drop a `YYYYMMDDHHMMSS_description.sql` file (Goose convention with `-- +goose Up` / `-- +goose Down` markers) into `internal/database/migrations/sql/`. The `//go:embed` pattern picks it up at compile time — no Go code change needed. `database.Migrate` runs the up direction on every successful boot when the database subsystem is enabled.
- **New domain model**: add a Bun struct type in a new file under `internal/database/orm/`; access the connection via `orm.Database()`. Don't write directly to `database.Connection()` from app code — go through the ORM so dialect changes stay isolated.
- **New health component**: register a `health.CheckFunc` against `app.HealthRegistry()` from inside `Application.registerHealthChecks` (preferred — keeps lifecycle ownership of registration), or from a subcommand if the dependency lives outside the `internal/application/` package. Closures may be invoked frequently, so keep them sub-second; put expensive probes behind a cached background poller. Conditionally register subsystems that may be disabled — only active components should appear in the report.
- **Release behaviour change**: edit `.releaserc.json` for semantic-release plugin config (commit-analyzer rules, release notes preset, exec hooks, branch channels, GitHub asset list). Cross-check that any new binary or other artifact added to `bin/` is also listed under `@semantic-release/github` `assets` so it ships with the release, AND that the `subject-path` glob in `.github/workflows/800-call-semantic-release.yaml`'s `Attest provenance for file artifacts` step includes it — every release artifact must carry a GitHub-signed Sigstore attestation. Add an `actions/attest-sbom` step too if the artifact has a corresponding SBOM.

## Supply-chain signing policy

This project signs **hashes, not artifacts**. `task sign` produces only `<binary>.sha256.asc` (never `<binary>.asc`). SBOMs are themselves manifests and ARE signed directly (`sbom.json.asc`, `container-sbom.json.asc`). The signed hash transitively pins the artifact. Don't add bare `<binary>.asc` entries to the release asset list or to any new signing target. In addition, every release artifact and the OCI subject (container image) is attested via `actions/attest-build-provenance` and (for SBOMs) `actions/attest-sbom`. Verification path going forward: `gh attestation verify <artifact> --owner servercurio` for primary verification, GPG-on-hashes for offline-capable secondary verification.

## Things to leave alone unless asked

- This repo uses **Cobra without Viper** — config layering goes through `internal/config` + `internal/env`, not Viper. Don't pull Viper in.
- **ants/v2** is the only goroutine-pool library. Don't add a second.
- `obfusicate/` exposes one tiny helper (`ConcealPrefix`) used for log redaction — leave it alone unless the task is explicitly about redaction; the misspelled package name is intentional and matches the import path.
