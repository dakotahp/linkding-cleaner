# linkding-cleaner

![linkding-cleaner](docs/images/banner.png)

A CLI tool that checks all your [linkding](https://github.com/sissbruecker/linkding) bookmarks
for broken URLs and archives any that return 404.

## Usage

```sh
linkding-cleaner --url https://links.example.com
```

Required:

| Flag / Env var | Description |
|----------------|-------------|
| `--url` | Base URL of your linkding instance |
| `--token` or `LINKDING_TOKEN` | Your linkding API token |

Optional:

| Flag | Default | Description |
|------|---------|-------------|
| `--concurrency` | `10` | Number of parallel URL checks |
| `--timeout` | `10s` | Per-request timeout |
| `--dry-run` | | Check URLs and report what would be archived without making changes |
| `--version` | | Print version and exit |

## Examples

Using an environment variable for the token:

```sh
export LINKDING_TOKEN=your-token-here
linkding-cleaner --url https://links.example.com
```

Passing the token inline (useful in scripts):

```sh
linkding-cleaner --url https://links.example.com --token your-token-here
```

Tune concurrency and timeout for slower connections:

```sh
linkding-cleaner --url https://links.example.com --concurrency 5 --timeout 30s
```

## Install

**Homebrew (macOS):**

```sh
brew install dakotahp/tap/linkding-cleaner
```

**Go install:**

```sh
go install github.com/dakotahp/linkding-cleaner/cmd/linkding-cleaner@latest
```

**Download a binary** from the [releases page](https://github.com/dakotahp/linkding-cleaner/releases).

**Build from source:**

```sh
git clone https://github.com/dakotahp/linkding-cleaner
cd linkding-cleaner
go build ./cmd/linkding-cleaner
```

## Releasing

Releases are automated with [release-please](https://github.com/googleapis/release-please) and [GoReleaser](https://goreleaser.com).

1. Each push to `master` updates an open release PR. It holds the next version number and the new `CHANGELOG.md` entries, both taken from the conventional commit messages (`feat:`, `fix:`, and so on).
2. Merging the release PR tags the version and creates a GitHub Release with those notes.
3. GoReleaser then builds binaries for Linux, macOS, and Windows (amd64 + arm64), uploads them with `checksums.txt`, and updates the [Homebrew tap](https://github.com/dakotahp/homebrew-tap) formula.

While the version is below 1.0, `feat:` bumps the minor version and `fix:` bumps the patch version. A breaking change (`feat!:`) also bumps only the minor version.

The workflow needs a `HOMEBREW_TAP_GITHUB_TOKEN` repository secret: a personal access token with write access to the `homebrew-tap` repo.

## Output

- Red `[404]` — broken link, automatically archived in linkding
- Yellow `[403]` — forbidden (not archived)
- Plain `[200]` — healthy
- `[ERR]` — network error or timeout (skipped)

When run interactively, an animated progress bar and live elapsed time are shown during checking. Total elapsed time is always printed at the end.
