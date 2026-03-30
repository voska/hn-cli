# hn

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Agent-friendly CLI for [Hacker News](https://news.ycombinator.com) via the [Algolia API](https://hn.algolia.com). Compact plaintext output by default, structured JSON with `--json`. No authentication required.

## Installation

```bash
go install github.com/voska/hn-cli/cmd/hn@latest
```

Or build from source:

```bash
git clone https://github.com/voska/hn-cli.git
cd hn-cli
make build    # outputs to bin/hn
make install  # copies to $GOPATH/bin
```

## Usage

### Search

```bash
hn search "local LLMs"                    # Search stories
hn search "local LLMs" --comments         # Search comments
hn search "local LLMs" --sort date        # Sort by date
hn search "local LLMs" -n 5              # Limit results
hn search "local LLMs" --after 2026-01-01 # Date filter
hn search "local LLMs" --min-points 100   # Minimum points
```

### Front Page

```bash
hn front          # Current front page
hn front -n 10    # Fewer stories
```

### Read

```bash
hn read 12345678            # Story + top comments (3 levels)
hn read 12345678 --expand   # All comments expanded
```

### User

```bash
hn user pg        # Profile, karma, about
```

### Other

```bash
hn status         # API health check with latency
hn version        # Version, commit, build date
hn --version      # Short version
```

## Output

Compact plaintext to stdout, errors to stderr. All commands support `--json` for structured output.

```
$ hn search "Claude Code" -n 2
5632 results:
1. [2127pts 963c] Claude 3.7 Sonnet and Claude Code  @bakugo  1y ago
   https://www.anthropic.com/news/claude-3-7-sonnet
   HN: https://news.ycombinator.com/item?id=43163011
2. [1298pts 565c] Cowork: Claude Code for the rest of your work  @adocomplete  2mo ago
   https://claude.com/blog/cowork-research-preview
   HN: https://news.ycombinator.com/item?id=46593022
```

Aliases: `s` (search), `f` (front), `r` (read), `u` (user).

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 3 | Empty result set |
| 5 | Not found (invalid ID/username) |
| 8 | Retryable error (network/timeout) |

## API

Uses the [HN Algolia API](https://hn.algolia.com/api/v1/) -- free, public, no authentication or API keys required.

## License

[MIT](LICENSE)
