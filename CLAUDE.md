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

CI uses `golangci-lint run --enable errcheck`. Run it locally with golangci-lint installed. CI reads the Go version from `go.mod`, so bumping the `go` directive also bumps CI.

## Architecture

The binary lives at `cmd/linkding-cleaner/main.go`. All logic flows through `run(args, stdout)`, which is tested directly in `main_test.go` without subprocess overhead.

Three internal packages with clear single responsibilities:

- `internal/linkding` — REST client for the linkding API. Fetches all bookmark pages in parallel using `errgroup`, archives bookmarks via POST.
- `internal/checker` — URL liveness checks. Sends HEAD first, retries with GET on 405.
- `internal/reporter` — Writes coloured status lines to an `io.Writer` (red 404, yellow 403, plain otherwise).

`main.go` wires these together: fetch all bookmarks → fan out URL checks with a semaphore-bounded worker pool → archive any 404s.

All tests use `httptest.NewServer` — no mocks, no interfaces. The linkding client and main entrypoint both accept an `*http.Client` or an `io.Writer` so real HTTP servers can be injected in tests.

## Releasing

release-please (`.github/workflows/release.yml`) keeps a release PR open on `master`. Merging it tags the version, writes `CHANGELOG.md`, and creates the GitHub Release. The same workflow then runs GoReleaser, which uploads Linux/macOS/Windows × amd64/arm64 binaries and checksums to that release and pushes a Homebrew cask to `dakotahp/homebrew-tap`. GoReleaser does not write release notes (`release.mode: keep-existing`). Do not tag versions by hand.

Commit messages must be conventional commits (`feat:`, `fix:`, `chore:`, ...) because release-please builds the version bump and changelog from them. PRs land as merge commits; release-please ignores the merge commit's own message.
