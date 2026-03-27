# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build ./...               # build all packages
go test ./...                # run all tests
go test ./internal/linkding/ # run tests for a single package
go test -run TestFetchPage_SinglePage ./internal/linkding/  # run a single test
gofmt -l .                   # list files needing formatting
gofmt -w .                   # format all files
go vet ./...                 # vet all packages
```

CI uses `golangci-lint run --enable errcheck`. Run it locally with golangci-lint installed. The `go` directive in `go.mod` is pinned to `1.24.0` intentionally — golangci-lint's pre-built binary requires it; do not bump it with `go mod tidy`.

## Architecture

The binary lives at `cmd/linkding-cleaner/main.go`. All logic flows through `run(args, stdout)`, which is tested directly in `main_test.go` without subprocess overhead.

Three internal packages with clear single responsibilities:

- `internal/linkding` — REST client for the linkding API. Fetches all bookmark pages in parallel using `errgroup`, archives bookmarks via POST.
- `internal/checker` — URL liveness checks. Sends HEAD first, retries with GET on 405.
- `internal/reporter` — Writes coloured status lines to an `io.Writer` (red 404, yellow 403, plain otherwise).

`main.go` wires these together: fetch all bookmarks → fan out URL checks with a semaphore-bounded worker pool → archive any 404s.

All tests use `httptest.NewServer` — no mocks, no interfaces. The linkding client and main entrypoint both accept an `*http.Client` or an `io.Writer` so real HTTP servers can be injected in tests.

## Releasing

Tag a version to trigger GoReleaser via GitHub Actions:

```bash
git tag v0.2.0 -a -m "v0.2.0"
git push origin v0.2.0
```

GoReleaser builds Linux/macOS/Windows × amd64/arm64, publishes a GitHub Release with checksums, and pushes a Homebrew formula to `dakotahp/homebrew-tap`. The workflow requires a `HOMEBREW_TAP_GITHUB_TOKEN` secret with write access to that repo.
