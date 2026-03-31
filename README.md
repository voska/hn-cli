# hn — Hacker News CLI

[![CI](https://github.com/voska/hn-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/voska/hn-cli/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/voska/hn-cli)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Agent-friendly CLI for [Hacker News](https://news.ycombinator.com) via the [Algolia API](https://hn.algolia.com). Data goes to stdout (parseable), hints/progress to stderr. No authentication required.

```bash
$ hn search "local LLMs" -n 3
5632 results:
1. [2127pts 963c] Running Local LLMs Is a Waste of Time  @dang  3mo ago
   https://example.com/local-llms
   HN: https://news.ycombinator.com/item?id=43163011
2. [891pts 412c] Ollama: Run LLMs Locally  @thunderbong  8mo ago
   https://ollama.ai
   HN: https://news.ycombinator.com/item?id=42591244
3. [654pts 287c] How I Run LLMs on a Raspberry Pi  @tosh  1y ago
   https://example.com/rpi-llms
   HN: https://news.ycombinator.com/item?id=41023547

$ hn front -n 3
1. [312pts 142c] Show HN: I built a CLI for everything  @pg  2h ago
   https://example.com/cli
2. [198pts  87c] Why SQLite Is So Great for the Edge  @dang  4h ago
   https://example.com/sqlite
3. [145pts  53c] The death of microservices  @mfiguiere  6h ago
   https://example.com/microservices

$ hn read 43163011 --json | jq '.comments | length'
963
```

Run `hn --help` for the full command tree.

## Install

**Homebrew** (macOS / Linux):

```bash
brew install voska/tap/hn
```

**Go**:

```bash
go install github.com/voska/hn-cli/cmd/hn@latest
```

**Binary**: download from [Releases](https://github.com/voska/hn-cli/releases).

## Quick Start

```bash
# Search stories
hn search "distributed systems" --min-points 100

# Search comments
hn search "Go vs Rust" --comments

# Recent stories sorted by date
hn search "WASM" --sort date --after 2026-01-01

# Current front page
hn front

# Read a story with comments (3 levels deep)
hn read 43163011

# Read with all comments expanded
hn read 43163011 --expand

# User profile
hn user pg

# API health check
hn status
```

## Agent Skill

Install as a [Claude Code skill](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/skills) for AI-assisted HN research:

```bash
npx skills add -g voska/hn-cli
```

## API

Uses the [HN Algolia API](https://hn.algolia.com/api/v1/) -- free, public, no authentication or API keys required.

## Output Modes

| Flag | Description |
|------|-------------|
| (default) | Compact plaintext to stdout |
| `--json` | Structured JSON to stdout |

## Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `search <query>` | `s` | Search stories or comments |
| `front` | `f` | Current front page stories |
| `read <id>` | `r` | Story with threaded comments |
| `user <username>` | `u` | User profile and karma |
| `status` | | API health check with latency |
| `version` | | Version, commit, build date |

All commands support `--json`.

## Exit Codes

| Code | Name | Meaning |
|------|------|---------|
| 0 | success | Operation completed |
| 1 | error | General error |
| 3 | empty | No results found |
| 5 | not_found | Invalid ID or username |
| 8 | retryable | Transient error, safe to retry |

## Development

```bash
make build    # Build to bin/hn
make test     # Run tests with race detector
make lint     # Run linter
make fmt      # Format code
```

## License

MIT
