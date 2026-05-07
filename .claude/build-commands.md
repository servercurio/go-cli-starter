# Key Build Commands

- `task` — Default pipeline: clean → vendor → lint → build all platform binaries.
- `task lint` — Runs `go fmt`, a strict `go vet` (atomic, defers, assign, bools, buildtag, framepointer, lostcancel, loopclosure, nilfunc, shift, stdmethods, stringintconv, structtag), and `golangci-lint run` against `.golangci.yml`. The golangci-lint binary is installed via `go install` on the fly (pinned version inside the task), so no system-level prerequisite.
- `task test` — Runs `go test -parallel 4 -cover -coverprofile cover.out -race -v ./...` (with `CGO_ENABLED=1` for the race detector). The `cover.out` profile is written into the repo root and removed by `task clean`.
- `task vendor` — Runs `go mod tidy` followed by `go mod vendor`. Use after any `go.mod` change.
- `task generate` — Runs `go generate ./...`. Required after a fresh clone (and run automatically by `task build`) because `internal/version/commit.txt` is gitignored but is `//go:embed`-ed.
- `task build` — Cross-compiles 6 binaries (linux/darwin/windows × amd64/arm64) into `bin/`. Calls `task generate` first.
- `task hash` — Writes one `bin/<binary>.sha256` file per built binary (`shasum -c` compatible). Run locally to verify a build (`cd bin && shasum -a 256 -c appcli-linux-amd64.sha256`).
- `task sign` — Depends on `task hash`. GPG-signs **only** each `.sha256` file (producing `bin/<binary>.sha256.asc`). Binaries themselves are NOT signed — the signed hash transitively pins the binary, and consumers verify with `gpg --verify bin/appcli-linux-amd64.sha256.asc bin/appcli-linux-amd64.sha256 && shasum -a 256 -c bin/appcli-linux-amd64.sha256`. Requires `gpg` with a configured signing key.
- `task sbom` — Generates CycloneDX 1.5 SBOMs at `bin/sbom.json` and `bin/sbom.xml` via `cyclonedx-gomod`, then GPG-signs each output (`bin/sbom.{json,xml}.asc`). Listed in `.releaserc.json`'s GitHub assets and run by `publishCmd`, so each release ships both flavours plus their signatures.
- `task run:serve` — Builds for the current platform and runs `appcli serve` (the daemon example). Ctrl-C stops it.
- `task run:copy` — Builds for the current platform and runs `appcli copy README.md /tmp/copy.md` (the one-shot example).
- `task build:container` / `task run:container` — Build / run the Docker image (local single-arch). `run:container` invokes the bare ENTRYPOINT with `--help`.
- `task container:build:multiarch` — Multi-arch (linux/amd64,linux/arm64) buildx push to `ghcr.io/servercurio/go-cli-starter`. Reads version from `internal/version/version.txt`. Used by the release pipeline; expects the caller to have run `docker login ghcr.io` first.
- `task container:sbom` / `task container:sbom:sign` — Generate `bin/container-sbom.{json,xml}` for the published container image via `syft`, then GPG-sign each.
- `task clean` — Removes `bin/`, `dist/`, and coverage output.
