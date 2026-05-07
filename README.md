<p align="center">
  <img src="docs/logo.svg" alt="Server Curio — Project Templates" width="600">
</p>

# go-cli-starter

A production-ready Go starter template for **command-line tools and CLI daemons**. It bundles a Cobra-driven command tree, structured logging, layered configuration, an optional PostgreSQL/Bun ORM, a shared goroutine pool, and a hardened release pipeline (signed hashes, SBOMs, Sigstore attestations) — without imposing choices about the actual work your CLI does.

The compiled binary is named `appcli`. Two example subcommands ship out of the box:

- `appcli serve` — long-running daemon with graceful shutdown on `SIGINT` / `SIGTERM`
- `appcli copy SRC DST [--recursive --workers N]` — one-shot file/tree copier that fans recursive copies through the shared goroutine pool

Replace them with your own subcommands and you have a new CLI.

## Table of contents

- [Features](#features)
- [Requirements](#requirements)
- [Quick start](#quick-start)
- [Project layout](#project-layout)
- [Configuration](#configuration)
  - [Config files](#config-files)
  - [Environment variables](#environment-variables)
  - [Flag overlay](#flag-overlay)
- [Subcommands](#subcommands)
  - [`serve` (daemon example)](#serve-daemon-example)
  - [`copy` (one-shot example)](#copy-one-shot-example)
  - [`version`](#version)
- [Goroutine pool](#goroutine-pool)
- [Database (optional)](#database-optional)
- [Adding a subcommand](#adding-a-subcommand)
- [Build tasks](#build-tasks)
- [Container](#container)
- [Releases](#releases)
  - [Release artifacts](#release-artifacts)
  - [Verifying a release](#verifying-a-release)
  - [Required repository secrets](#required-repository-secrets)
- [License](#license)

## Features

- **Cobra command tree** — composable subcommands with POSIX-style flags, automatic help, shell-completion generation
- **Layered configuration** with documented precedence: defaults → YAML/JSON config files → environment variables (prefix `APP_`) → CLI flags
- **Structured logging** via [zerolog](https://github.com/rs/zerolog), wired to both daemons and one-shot tools
- **Graceful shutdown** on `SIGINT`, `SIGTERM`, and `SIGUSR1` (Unix) / `SIGINT` (Windows) for daemon subcommands
- **Shared goroutine pool** backed by [panjf2000/ants](https://github.com/panjf2000/ants) — bounds concurrency across every subsystem with one knob (`--workers` / `APP_POOL_SIZE`)
- **Optional database subsystem** — PostgreSQL via `pgx`, Bun ORM, Goose migrations; empty DSN disables it entirely
- **Health registry** — Spring Boot Actuator-style component model, snapshot-able from any subcommand
- **Errorx-categorised errors** — typed namespaces under `internal/errors/` instead of ad-hoc `errors.New`
- **Cross-platform builds** for `linux`, `darwin`, `windows` × `amd64`, `arm64`
- **Universal supply-chain attestations** — every release artifact and OCI subject (binaries, hash files, container image, all CycloneDX SBOMs in JSON and XML) carries a GitHub-signed Sigstore attestation verifiable with `gh attestation verify`

## Requirements

- Go 1.26+
- [Task](https://taskfile.dev) (canonical build runner)
- Docker (optional, for container builds)

## Quick start

```sh
# Vendor, lint, and build all platform binaries
task

# Print help (lists every subcommand)
./bin/appcli-darwin-arm64 --help

# Run the daemon locally (builds first); Ctrl-C to stop
task run:serve

# Run the one-shot example (copies README.md to /tmp/copy.md)
task run:copy

# Run all tests with race detector and coverage
task test
```

## Project layout

```
cmd/appcli/          # main package — Cobra Execute() shim
internal/
  application/       # Application lifecycle (Configure → Initialize → Run/RunUntilSignal)
  cli/               # Cobra command tree (root, serve, copy, version) + flag overlay
  pool/              # Goroutine pool wrapping panjf2000/ants
  config/            # YAML/JSON config-file loader with multi-path search
  env/               # Type-safe environment-variable parsers
  logging/           # zerolog-backed Daemon logger + startup notifications
  errors/            # joomcode/errorx namespaces (FileSystem, Database, Pool)
  health/            # Per-component health registry + Report model
  database/          # Optional PostgreSQL connection pool + Goose migrations
  database/orm/      # Optional Bun ORM singleton
  obfusicate/        # Credential masking for log redaction
  version/           # Build-time version metadata (commit, semver, tag)
docs/                # Non-Go documentation assets (logo.svg)
.github/workflows/   # CI: code-compiles, unit-test, vulncheck, semantic-release
```

## Configuration

Configuration flows through four ordered sources, with later sources overriding earlier ones:

1. **Defaults** — `application.DefaultConfig()` populates every subsystem with sensible values.
2. **Config file** — first hit in `/etc/appcli/`, `~/.config/appcli/`, then the working directory; YAML or JSON, base name `appcli`. Use `--config <path>` to bypass the search.
3. **Environment variables** — every field is hydrated under the `APP_` prefix (e.g. `APP_DAEMON_LOG_LEVEL`, `APP_DATABASE_DSN`, `APP_POOL_SIZE`).
4. **CLI flags** — explicitly-set persistent flags overlay onto the loaded config in each subcommand's `PreRunE`. Defaulted (unset) flags are *not* applied, so they can't clobber file/env values.

### Config files

Drop an `appcli.yaml` next to the binary or in `~/.config/appcli/`:

```yaml
logging:
  daemon:
    enabled: true
    level: info
    prettyPrint: true
    includeCaller: false

database:
  driver: pgx
  dsn: postgres://user:pass@localhost:5432/appcli?sslmode=disable
  maxOpenConns: 25
  maxIdleConns: 5

pool:
  size: 16
  expiryDuration: 1m
  nonBlocking: false
```

### Environment variables

Every field is hydrated under the `APP_` prefix. Defaults come from each subsystem's `DefaultConfig()`; an unset variable leaves the resolved value at its file or default.

#### Logging (`APP_DAEMON_LOG_*`)

| Variable                          | Default | Description                                                                                                       |
| --------------------------------- | ------- | ----------------------------------------------------------------------------------------------------------------- |
| `APP_DAEMON_LOG_ENABLED`          | `true`  | Toggle the daemon logger. `false` swaps in `zerolog.Nop()` so callers can keep emitting events without overhead.  |
| `APP_DAEMON_LOG_LEVEL`            | `info`  | Verbosity. One of `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic` (parsed by `zerolog.ParseLevel`).   |
| `APP_DAEMON_LOG_PRETTY_PRINT`     | `true`  | Render via zerolog's `ConsoleWriter` (coloured, human-readable). Set `false` for newline-delimited JSON output.   |
| `APP_DAEMON_LOG_INCLUDE_CALLER`   | `false` | Append the caller's `file:line` to every event. Useful in development; adds runtime cost in tight log loops.      |

#### Database (`APP_DATABASE_*`)

The database subsystem is opt-in. Leave `APP_DATABASE_DSN` empty (the default) and `Application.Initialize` skips connection setup, migrations, and the ORM entirely.

| Variable                          | Default      | Description                                                                                                                          |
| --------------------------------- | ------------ | ------------------------------------------------------------------------------------------------------------------------------------ |
| `APP_DATABASE_DRIVER`             | `pgx`        | `database/sql` driver name. PostgreSQL via pgx ships in the starter; other drivers require importing them in `internal/database/`.   |
| `APP_DATABASE_DSN`                | _(empty)_    | Connection string. **Empty disables the database subsystem entirely** (Connect/Migrate become no-ops; no `database` health check).   |
| `APP_DATABASE_MAX_OPEN_CONNS`     | `25`         | Max open connections in the pool. Zero or negative means unlimited (matches `database/sql` default).                                 |
| `APP_DATABASE_MAX_IDLE_CONNS`     | `5`          | Max idle connections retained in the pool. Lower than `MAX_OPEN_CONNS` so the pool can shed cold connections.                        |
| `APP_DATABASE_CONN_MAX_LIFETIME`  | `1h`         | Max lifetime of a connection before recycling. Useful for credential rotation or proxies that drop long-lived connections.           |
| `APP_DATABASE_CONN_MAX_IDLE_TIME` | `5m`         | Max time an idle connection may sit in the pool before being closed.                                                                 |

#### Pool (`APP_POOL_*`)

| Variable                          | Default                | Description                                                                                                                                                          |
| --------------------------------- | ---------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `APP_POOL_SIZE`                   | `runtime.NumCPU() × 2` | Maximum concurrently-running goroutines. A value `<= 0` means unlimited (ants treats this as `math.MaxInt32`). The `--workers` flag overlays this.                   |
| `APP_POOL_NON_BLOCKING`           | `false`                | Rejection-on-full behaviour. When `true`, `Submit` returns `PoolExhausted` instead of blocking the caller — set for latency-sensitive paths that prefer to shed load. |
| `APP_POOL_EXPIRY_DURATION`        | `1m`                   | Idle-worker reap interval. `time.ParseDuration` syntax (`30s`, `5m`, `1h`).                                                                                          |
| `APP_POOL_PRE_ALLOC`              | `false`                | When `true`, ants allocates the worker queue at construction instead of growing on demand. Useful when `SIZE` is large and known up front.                           |
| `APP_POOL_MAX_BLOCKING_TASKS`     | `0`                    | Bounds the number of callers that may block in `Submit` when the pool is full and `NON_BLOCKING=false`. `0` (the default) means unbounded blocking.                  |

### Flag overlay

The root command exposes a small set of persistent flags that overlay onto the loaded config:

| Flag           | Overlays                          |
| -------------- | --------------------------------- |
| `--config`     | Config-file search path           |
| `--log-level`  | `logging.daemon.level`            |
| `--log-pretty` | `logging.daemon.prettyPrint`      |
| `--db-dsn`     | `database.dsn`                    |
| `--workers`    | `pool.size`                       |

## Subcommands

### `serve` (daemon example)

```sh
appcli serve
```

Initialises the database (if a DSN is configured), the goroutine pool, and the health registry; logs a `daemon ready` event with pool stats; then blocks until a shutdown signal arrives. Replace the body in `internal/cli/serve.go` with your own loop.

### `copy` (one-shot example)

```sh
# Single file
appcli copy README.md /tmp/copy.md

# Recursive directory copy via the goroutine pool
appcli copy --recursive --workers 8 ./src /tmp/dst
```

`--recursive` walks the source tree and submits each file copy through `app.Pool().SubmitWithContext(...)`. The `--workers` flag bounds concurrency by overriding `pool.size`.

### `version`

```sh
appcli version
# v1.2.3 (a1b2c3d…)
```

Bypasses config loading and database initialisation — safe to invoke against a binary whose config might be invalid.

## Goroutine pool

The pool is a thin wrapper around [panjf2000/ants/v2](https://github.com/panjf2000/ants). `Application` constructs one at `Initialize` time and exposes it via `app.Pool()`. Subcommands fan out work through it instead of constructing their own pools, so a single knob (`--workers` / `APP_POOL_SIZE`) bounds total in-flight goroutines:

```go
p := app.Pool()

// Cancellation-aware submit — worker checks ctx.Err() before invoking the body.
err := p.SubmitWithContext(ctx, func(ctx context.Context) {
    doWork(ctx)
})
if err != nil {
    // errorx-categorised: pool.PoolExhausted | pool.PoolReleased | pool.SubmitFailed
}
```

Submit errors come pre-wrapped in `internal/errors`'s `PoolErrors` namespace so callers can branch via `errorx.IsOfType` without importing `ants`.

## Database (optional)

The database subsystem is **opt-in**. Leave `database.dsn` empty (the default) and `Application.Initialize` skips connection setup, migrations, and the ORM. To enable:

1. Set `APP_DATABASE_DSN` (or `--db-dsn`) to a valid PostgreSQL connection string.
2. Drop a `YYYYMMDDHHMMSS_<description>.sql` migration with `-- +goose Up` / `-- +goose Down` markers into `internal/database/migrations/sql/`. The `//go:embed` pattern picks it up at compile time.
3. Add Bun struct types under `internal/database/orm/` and reach for the connection via `orm.Database()`.

Migrations run on every successful boot (`database.Migrate`).

## Adding a subcommand

1. Create a new file under `internal/cli/` returning a `*cobra.Command`. Wire any subcommand-specific flags to a local `*flags` struct.
2. Add it to the slice in `NewRootCommand` (in `internal/cli/root.go`).
3. Inside `RunE`, reach for `rc.app` (already constructed and configured by `PersistentPreRunE`); call `app.Initialize()` then either `app.Run(ctx, body)` (one-shot) or `app.RunUntilSignal(ctx, body)` (daemon).

That's it. Help text, completion, env-var hydration, and config loading all flow automatically from the existing infrastructure.

## Build tasks

Common targets (see `Taskfile.yaml` for the full list):

| Task | Purpose |
| ---- | ------- |
| `task` | Default pipeline: clean → vendor → lint → build all platforms |
| `task lint` | `go fmt` + strict `go vet` + `golangci-lint` |
| `task test` | `go test -race -cover ./...` |
| `task vendor` | `go mod tidy && go mod vendor` |
| `task generate` | Run all `go:generate` directives (version metadata) |
| `task build` | Cross-compile 6 binaries (linux/darwin/windows × amd64/arm64) into `bin/` |
| `task hash` | Per-binary SHA-256 hash files |
| `task sign` | GPG-sign each `.sha256` (binaries themselves are not signed — the signed hash transitively pins the binary) |
| `task sbom` | CycloneDX SBOM (JSON + XML) via `cyclonedx-gomod`, GPG-signed |
| `task run:serve` | Build and run `appcli serve` locally |
| `task run:copy` | Build and run `appcli copy README.md /tmp/copy.md` |
| `task build:container` / `task run:container` | Local single-arch container build / run |
| `task container:build:multiarch` | Multi-arch buildx push to `ghcr.io/servercurio/go-cli-starter` (release pipeline) |
| `task clean` | Remove `bin/`, `dist/`, and coverage output |

## Container

`task build:container` builds a single-arch image; `task container:build:multiarch` (used by the release pipeline) pushes a multi-arch manifest to GHCR. The container's entrypoint is the bare binary, so any Cobra subcommand works directly:

```sh
docker run --rm ghcr.io/servercurio/go-cli-starter:latest --help
docker run --rm ghcr.io/servercurio/go-cli-starter:latest serve
docker run --rm -v /tmp:/tmp ghcr.io/servercurio/go-cli-starter:latest copy /tmp/in.txt /tmp/out.txt
```

## Releases

Releases are produced by `100-user-deploy-release-artifact.yaml` (manual `workflow_dispatch`), which delegates to the reusable `800-call-semantic-release.yaml`. The release flow:

1. Runs `semantic-release` to compute the next version from conventional-commit messages.
2. Writes the resolved version to `internal/version/version.txt` (consumed by `//go:embed`).
3. Commits the version bump back to the release branch with a `chore(release): X.Y.Z [skip ci]` message. The commit is **GPG-signed** (DCO `Signed-off-by` is appended automatically by a per-clone `prepare-commit-msg` hook installed in CI).
4. Runs `task build` (six platform binaries) → `task hash` + `task sign` (per-binary SHA256 files and GPG signatures of the hashes) → `task sbom` (Go module CycloneDX SBOM, JSON + XML, GPG-signed) → `task container:build:multiarch` (multi-arch buildx push to `ghcr.io/servercurio/go-cli-starter`) → `task container:sbom` + `task container:sbom:sign` (container CycloneDX SBOM, JSON + XML, GPG-signed).
5. Tags the commit and creates a GitHub Release. Every file artifact is uploaded as a release asset; the container image lives in GHCR.
6. Issues GitHub-signed Sigstore attestations via `actions/attest-build-provenance` (covering every release file plus the OCI subject) and `actions/attest-sbom` (binding each CycloneDX SBOM to its subjects). OCI attestations are pushed alongside the manifest in GHCR.

### Release artifacts

| Artifact class | Files / subjects | Notes |
| -------------- | ---------------- | ----- |
| **Binaries** | `appcli-{linux,darwin,windows}-{amd64,arm64}` (6 total) | No per-binary GPG signature — verify via the signed hash file |
| **Hash files** | `<binary>.sha256` + `<binary>.sha256.asc` | `.asc` is the detached GPG signature of the hash file |
| **Container image** | `ghcr.io/servercurio/go-cli-starter:<version>` and `:latest` | Multi-arch manifest list (linux/amd64 + linux/arm64); attestations pushed to GHCR |
| **Go SBOM** | `sbom.json` + `sbom.xml` + `.asc` companions | CycloneDX, generated by `cyclonedx-gomod` |
| **Container SBOM** | `container-sbom.json` + `container-sbom.xml` + `.asc` companions | CycloneDX, generated by `syft` against the published image |
| **Attestations** | One `attest-build-provenance` per release file + one for the OCI subject; one `attest-sbom` per (SBOM format, subject) pair | Stored in the GitHub attestation log and (for the OCI subject) alongside the manifest in GHCR |

### Verifying a release

The recommended path is `gh attestation verify`, which works for both file artifacts and the OCI subject without any pre-shared key:

```sh
# Binary
gh attestation verify appcli-linux-amd64 --owner servercurio

# Hash file (proves universal coverage of every released file)
gh attestation verify appcli-linux-amd64.sha256 --owner servercurio

# Go SBOM
gh attestation verify sbom.json --owner servercurio

# Container image (provenance + container SBOM, fetched from GHCR)
gh attestation verify oci://ghcr.io/servercurio/go-cli-starter:<X.Y.Z> --owner servercurio
```

For offline-capable verification, use the GPG-signed hash files. **Binaries themselves are not GPG-signed** — only the `.sha256` files are. The signed hash transitively pins the binary, so:

```sh
# Verify the hash file was signed by the release key, then check the binary against it
gpg --verify appcli-linux-amd64.sha256.asc appcli-linux-amd64.sha256
shasum -a 256 -c appcli-linux-amd64.sha256
```

CycloneDX SBOM files (`sbom.json`, `container-sbom.json` and their `.xml` variants) are signed directly because the SBOM file is itself the manifest:

```sh
gpg --verify sbom.json.asc sbom.json
```

Branch policy (from `.releaserc.json`):

| Branch pattern   | Channel        | Notes                                                |
|------------------|----------------|------------------------------------------------------|
| `main`           | latest         | Default release branch                               |
| `release/X.Y`    | `X.Y.x`        | Maintenance branches; release range pinned to `X.Y.x` |
| `alpha/*`        | `alpha`        | Prerelease channel                                   |
| `beta/*`         | `beta`         | Prerelease channel                                   |
| `rc/*`           | `rc`           | Release-candidate channel                            |

Release rules (commit type → version bump):

| Type                | Bump  |
|---------------------|-------|
| `feat`              | minor |
| `fix`               | patch |
| `refactor`, `build` | patch |
| `BREAKING CHANGE` (footer or `!` in subject) | minor |
| `chore`, `ci`, `docs`, `style`, `test` | none  |

Run a dry run from the workflow dispatch UI by checking **Perform dry run** — semantic-release will print the version that *would* be released without tagging, committing, or publishing.

> **Note on protected branches**: the workflow uses the default `GITHUB_TOKEN`. If `main` is protected with restrictions that block the GitHub App from pushing back the version-bump commit, configure a PAT (e.g. `GH_ACCESS_TOKEN`) with bypass rights and pass it as `release-token` from `100-user-deploy-release-artifact.yaml`.

### Required repository secrets

| Secret             | Purpose                                                                  |
|--------------------|--------------------------------------------------------------------------|
| `GPG_PRIVATE_KEY`  | ASCII-armored private GPG key used to sign the release commit and every `.sha256` / SBOM file. Generate with `gpg --armor --export-secret-keys <KEY-ID>`. |
| `GPG_PASSPHRASE`   | Passphrase for the private GPG key.                                      |
| `GH_ACCESS_TOKEN`  | *(Optional.)* PAT used as `release-token` when `GITHUB_TOKEN` can't satisfy a constraint. The token must carry: `contents: write` (release commits and tags), `packages: write` (GHCR push for the container image), `id-token: write` (OIDC for Sigstore Fulcio), and `attestations: write`. The default `GITHUB_TOKEN` already has these scopes when the workflow declares the matching `permissions:` block; a PAT is only needed when branch protection blocks the App-bound default token. |
| `CODECOV_TOKEN`    | *(Optional, used by PR Checks.)* Codecov upload token; recommended for public repos to avoid rate limits, required for private repos. |

The corresponding **public key** must be added to the GitHub account/org used to publish releases (and shared with consumers) so signatures can be verified.

## License

See [LICENSE](LICENSE).
