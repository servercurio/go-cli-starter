# Contributing

Thanks for your interest in `go-cli-starter`. The repository is a starter template, so contributions are oriented toward keeping the scaffold clean, generic, and reusable rather than expanding feature scope.

## Before you start

- **Open an issue first** for anything beyond a typo or one-line bug fix. Discussing scope and approach saves rework.
- **Don't add downstream concerns.** Persistence drivers beyond the bundled pgx wiring, auth schemes, validation libraries, and domain models are intentionally out of scope. The template stays generic so consumers can layer their own choices.

## Local setup

```sh
git clone https://github.com/servercurio/go-cli-starter.git
cd go-cli-starter
task vendor
task
```

`task` is the canonical build runner. See `.claude/build-commands.md` for the full target list. Don't invoke `go build` / `go test` directly except for one-off debugging — go through the Taskfile so CI and local builds stay aligned.

## Required local git hook

Every clone needs a per-clone `prepare-commit-msg` hook that auto-appends a DCO `Signed-off-by:` line. The hook is **not** version-controlled; install it manually after cloning. Contents and rationale live in [`.claude/git-hooks.md`](.claude/git-hooks.md).

```sh
# Verify it's in place
ls -l .git/hooks/prepare-commit-msg
```

If the hook is missing, copy the script from `.claude/git-hooks.md` and `chmod +x` it.

## Commit-message conventions

This repo uses [Conventional Commits](https://www.conventionalcommits.org). The release flow (semantic-release) reads commit types to decide version bumps:

| Type                | Bump  | Example                                 |
|---------------------|-------|-----------------------------------------|
| `feat`              | minor | `feat: add /metrics endpoint`           |
| `fix`               | patch | `fix: handle nil DSN in IsHealthy`      |
| `refactor`, `build` | patch | `refactor: extract proxy IP helper`     |
| breaking change     | minor | `feat!: rename APP_OPENAPI_TITLE` (or `BREAKING CHANGE:` footer) |
| `chore`, `ci`, `docs`, `style`, `test` | none | `docs: clarify CORS default` |

Subject line: imperative mood, no trailing period, ≤72 chars. The `prepare-commit-msg` hook will append your DCO line automatically.

## Pull request workflow

1. Branch from `main` (or a `release/X.Y` branch for backports).
2. Run `task` locally before pushing — clean → vendor → lint → build of all six platform binaries.
3. Run `task test` and confirm coverage doesn't regress.
4. Open the PR. The "PR Checks" workflow runs `task lint`, `task test`, and `govulncheck`; the "PR Formatting" workflow validates the title against Conventional Commits.
5. Address review feedback in additional commits — don't force-push during review unless asked.
6. Squash on merge is fine; the squash subject should still be a valid Conventional Commit.

## Code conventions

See [`.claude/conventions.md`](.claude/conventions.md). Highlights:

- Functional options for builders (`module.WithRoutes(...)` etc.).
- `Application` lifecycle methods split into `application_<subsystem>.go` files; the base `application.go` only holds the struct and lifecycle skeleton.
- Errors use `joomcode/errorx` namespaces (`internal/errors/fs.go`).
- OpenAPI metadata lives next to handlers via `endpoint.With*` builder options — no comment-annotation grammars.

## Reporting security issues

Please **don't** file public issues for vulnerabilities. See [`SECURITY.md`](SECURITY.md) for the private reporting flow.
