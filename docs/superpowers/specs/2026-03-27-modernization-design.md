# linkding-cleaner Modernization Design

**Date:** 2026-03-27
**Status:** Approved

## Overview

Full modernization of a ~150-line Go CLI tool (originally written circa 2021) that checks
linkding bookmark URLs for 404s and archives broken ones. Goals: Go 1.24, zero third-party
deps, proper package structure for future growth, parallel execution, watertight test coverage.

---

## Package Structure

```
linkding-cleaner/
├── cmd/
│   └── linkding-cleaner/
│       └── main.go           # entry point: parse flags/env, wire packages, run
├── internal/
│   ├── linkding/
│   │   ├── client.go         # linkding API client: paginated fetch, archive
│   │   └── client_test.go
│   ├── checker/
│   │   ├── checker.go        # HTTP liveness checker: HEAD→GET fallback
│   │   └── checker_test.go
│   └── reporter/
│       ├── reporter.go       # colored terminal output
│       └── reporter_test.go
├── go.mod                    # module: linkding-cleaner, Go 1.24
├── go.sum
├── Makefile
└── README.md
```

`internal/` keeps packages private to the module. Each package has one job. `main.go` only
wires them together. `config.example.yml` is removed — no config file needed.

---

## Configuration & Secrets

No config file. Two inputs only:

| Input | Source | Priority |
|-------|--------|----------|
| Base URL | `--url` flag | required |
| API token | `--token` flag | 1st |
| API token | `LINKDING_TOKEN` env var | 2nd (fallback) |

If `--url` is missing or token is not provided via either source, print usage and `os.Exit(1)`.
No panics, no config file management, no YAML dependency.

Additional flags:
- `--concurrency` (default: 10) — max parallel URL checks
- `--timeout` (default: 10s) — per-request timeout for URL checks

---

## Dependencies

- `golang.org/x/sync/errgroup` — bounded concurrent work with error propagation (official Go team)
- Zero other third-party dependencies (Viper removed, no YAML library needed)

---

## Data Flow

```
main.go
  │  1. parse --url, --token/LINKDING_TOKEN, --concurrency, --timeout
  │  2. validate inputs, exit with usage on missing required values
  │
  ├─→ linkding.Client{BaseURL, Token}
  │     FetchAllBookmarks()
  │       ├─ fetch page 1 → read Count
  │       ├─ calculate remaining page offsets from Count + page size
  │       └─ fetch remaining pages in parallel → merge into []Bookmark
  │
  ├─→ worker pool (errgroup, --concurrency goroutines)
  │     for each Bookmark:
  │       checker.Check(url, timeout)
  │         ├─ HEAD request first (cheaper, no body)
  │         └─ GET fallback if HEAD returns 405 or errors
  │       reporter.Render(statusCode, bookmark)
  │       if 404 → linkding.Client.Archive(bookmark.ID)
  │
  └─→ exit 0
```

---

## Concurrency Design

**Page fetching:**
1. Use a fixed `pageSize = 100` constant, passed as `?limit=100` on every request
2. Fetch page 1 (offset 0) to get `count` (total bookmarks from API)
3. Calculate remaining page count: `ceil(count / 100) - 1` additional pages
4. Fetch all remaining pages in parallel via goroutines (each with its own `?offset=N&limit=100`)
5. Merge all results (order within pages preserved)

**URL checking:**
- Worker pool of `--concurrency` goroutines (default 10)
- Each worker: check URL → render result → archive if 404
- `errgroup` handles error propagation and goroutine lifecycle
- Results are printed as they arrive (not buffered to end)

---

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Missing `--url` or token | Print usage, `os.Exit(1)` |
| Network error on URL check | Log URL + error, skip bookmark, continue |
| HTTP timeout on URL check | Log URL + timeout, skip bookmark, continue |
| Archive request failure | Log bookmark + error, continue |
| Non-200/404/403 status | Report as-is (plain), no action |
| Linkding API error | Fatal error with message, `os.Exit(1)` |

No panics anywhere. All nil-resp risks from the original code are eliminated by checking errors
before using responses.

---

## HTTP Checker

The original code used `http.Get` for every URL. Modern approach:

1. Send `HEAD` request — cheaper (no body downloaded), respects the same auth/redirect rules
2. If server returns `405 Method Not Allowed` or any network error → retry with `GET`
3. Return final `(statusCode int, err error)`

All requests use `http.NewRequestWithContext` with a context carrying the `--timeout` deadline.

---

## Struct Naming

Original structs used non-idiomatic underscore names. Updated to Go conventions with JSON tags:

```go
// Before
type Bookmark struct {
    Id            int
    Website_title string
    Tag_names     []string
}

// After
type Bookmark struct {
    ID           int      `json:"id"`
    URL          string   `json:"url"`
    Title        string   `json:"title"`
    Description  string   `json:"description"`
    WebsiteTitle string   `json:"website_title"`
    TagNames     []string `json:"tag_names"`
}
```

---

## Testing Strategy

All tests use `httptest.NewServer` — real HTTP transport against a local fake server. No mocking
libraries, no interfaces required.

### `internal/linkding/client_test.go`
- Fake linkding API server returns canned JSON
- Test: single page (count fits in one page)
- Test: multi-page — verify all pages fetched, results merged correctly
- Test: auth header present on every request (`Authorization: Token <token>`)
- Test: archive sends correct POST to `/api/bookmarks/{id}/archive/`
- Test: API error response returns error, not panic

### `internal/checker/checker_test.go`
- Fake target server with configurable status code
- Test: 200 OK returned correctly
- Test: 404 returned correctly
- Test: 403 returned correctly
- Test: HEAD→GET fallback when HEAD returns 405
- Test: timeout respected (slow server, short timeout → error, no hang)

### `internal/reporter/reporter_test.go`
- Capture stdout
- Test: 404 output contains red ANSI code
- Test: 403 output contains yellow ANSI code
- Test: 200 output contains no ANSI color codes

### Integration (`cmd/linkding-cleaner/`)
- Wire all real packages against fake servers
- Test: full run with mixed 200/404/403 bookmarks across 2 pages → correct archives called

---

## CI Updates

Both GitHub Actions workflows updated:

```yaml
- uses: actions/checkout@v4      # was @v2
- uses: actions/setup-go@v5      # was @v1
  with:
    go-version: '1.24'            # was 1.16.5
```

Lint workflow: replace `wearerequired/lint-action@v1` (unmaintained) with
`golangci-lint-action@v6` running `gofmt` + `go vet` + `errcheck`.

---

## Module Name Fix

`go.mod` module name corrected from `linkdig-cleaner` (typo) to `linkding-cleaner`.

---

## Out of Scope

- Dry-run mode (no archiving, just report) — good future feature
- Output formats (JSON, CSV) — good future feature
- Delete (vs archive) broken bookmarks — good future feature
- Rate limiting / retry logic — good future feature
