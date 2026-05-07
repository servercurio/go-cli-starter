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

## Required commit signing

Every contributor commit must be **GPG-signed** in addition to carrying the DCO `Signed-off-by:` trailer above. The release workflow already produces signed `chore(release)` commits via an imported key (`.github/workflows/800-call-semantic-release.yaml`); contributor commits must match so the entire `main` history verifies.

Configure once per clone (or globally):

```sh
git config commit.gpgsign true
git config user.signingkey <YOUR-GPG-KEY-ID>
# Optional: also sign annotated tags
git config tag.gpgsign true
```

Verify a commit you just authored:

```sh
git log -1 --show-signature
# Look for "Good signature from ..." and a "G" in `git log --pretty='%G?'`
```

Don't bypass either requirement with `--no-gpg-sign` or `--no-verify`. If `gpg` prompts you for a passphrase on every commit, configure `gpg-agent` rather than disabling signing.

## Commit-message conventions

This repo uses [Conventional Commits](https://www.conventionalcommits.org). The release flow (semantic-release) reads commit types to decide version bumps:

| Type                | Bump  | Example                                 |
|---------------------|-------|-----------------------------------------|
| `feat`              | minor | `feat: add export subcommand`           |
| `fix`               | patch | `fix: handle nil DSN in IsHealthy`      |
| `refactor`, `build` | patch | `refactor: extract pool stats helper`   |
| breaking change     | minor | `feat!: rename APP_DAEMON_LOG_LEVEL` (or `BREAKING CHANGE:` footer) |
| `chore`, `ci`, `docs`, `style`, `test` | none | `docs: clarify pool sizing default` |

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

- **Cobra command tree, no Viper.** Config layering goes through `internal/config` + `internal/env`; flags overlay onto `*application.Config` in each command's `PersistentPreRunE`.
- New subcommands live under `internal/cli/`; register them in `internal/cli/root.go`'s `NewRootCommand`.
- `Application` lifecycle methods split into `application_<subsystem>.go` files; the base `application.go` only holds the struct and lifecycle skeleton.
- Errors use `joomcode/errorx` namespaces (`internal/errors/`).
- ants/v2 is the only goroutine-pool library — fan out work through `app.Pool()` rather than constructing per-subsystem pools.

## Reporting security issues

Please **don't** file public issues for vulnerabilities. See [`SECURITY.md`](SECURITY.md) for the private reporting flow.
