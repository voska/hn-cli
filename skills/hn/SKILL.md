---
name: hn
description: >-
  Hacker News CLI for searching stories, reading posts with comments, browsing
  the front page, and looking up users. Use when you need HN context: tech
  discussions, Show HN projects, Ask HN threads, startup launches, developer
  opinions, or trending topics. Free API, no auth required.
---

# hn -- Hacker News CLI

CLI for searching and reading Hacker News via the Algolia API. Returns compact
plaintext by default (token-efficient). No authentication required.

## Quick Reference

```bash
hn search "query"                    # Search stories (default)
hn search "query" --comments         # Search comments instead
hn search "query" --sort date        # Sort by date (default: relevance)
hn search "query" -n 10              # Limit results
hn search "query" --after 2026-01-01 # Only results after date
hn search "query" --min-points 100   # Minimum points filter

hn front                             # Front page in site rank order
hn front -n 10                       # Fewer results
hn front --feed show                 # top, new, best, ask, show or job

hn read ID                           # Story + top comments, capped
hn read ID -n 5 --replies 1          # Fewer comments
hn read ID --max-chars 300           # Shorter comments
hn read COMMENT_ID                   # Open a collapsed thread
hn read ID --expand                  # Every comment in full

hn user USERNAME                     # User profile and karma

hn status                            # API health check
hn version                           # Version info
```

Aliases: `hn s` = `hn search`, `hn f` = `hn front`, `hn r` = `hn read`, `hn u` = `hn user`.

## Workflow

1. Search or browse front page: `hn search "topic" -n 5` or `hn front`
2. Note the item ID from the HN URL in results
3. Read the full story + comments: `hn read ID`
4. Open a thread with `hn read COMMENT_ID`, using the `#ID` on any comment
5. Use `--expand` only when you need the whole thread: a big one costs tens of
   thousands of tokens

## Output Format

Compact plaintext by default (token-efficient). Use `--json` only when writing
to files -- never use `--json` in LLM context.

### Search output

```
1532 results:
1. [523pts 89c] Show HN: I built a tool for X  @username  2h ago
   https://example.com/article
   HN: https://news.ycombinator.com/item?id=12345
```

### Read output

Full story title, metadata, URL, body text, and comments in site rank order
with author, timestamp and `#ID`. By default 20 comments, 3 replies each, one
level of replies and 800 characters per comment. Cuts show as `[N more replies]`,
`[N more comments]` and `[+N chars]`.

### User output

```
@pg  karma:157316
Bug fixer.
https://news.ycombinator.com/user?id=pg
```

## Exit Codes

- `0` success
- `1` general error
- `3` empty result (no matches)
- `5` not found (invalid ID or username)
- `8` retryable error (network/timeout)

## Installation

```bash
go install github.com/voska/hn-cli/cmd/hn@latest
```
