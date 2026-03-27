# linkding-cleaner

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

```sh
go install linkding-cleaner/cmd/linkding-cleaner@latest
```

Or build from source:

```sh
git clone https://github.com/dakotahp/linkding-cleaner
cd linkding-cleaner
make build
```

## Output

- Red `[404]` — broken link, automatically archived in linkding
- Yellow `[403]` — forbidden (not archived)
- Plain `[200]` — healthy
- `[ERR]` — network error or timeout (skipped)
