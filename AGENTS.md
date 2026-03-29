# hn-cli

Go CLI for Hacker News via the Algolia API. No auth required.

## Build & Test

- `make build`: builds `bin/hn`
- `make install`: copies to `$GOPATH/bin/hn`
- `make fmt`: format Go files
- `make lint`: golangci-lint
- `make test`: unit tests with race detector
- `make ci`: fmt + lint + test + build

## Project Structure

- `cmd/hn/main.go` -- thin entrypoint, version embedding via ldflags
- `internal/cmd/root.go` -- all CLI commands (cobra)
- `internal/api/client.go` -- HN Algolia API client
- `internal/output/format.go` -- compact plaintext and JSON formatters
- `skills/hn/SKILL.md` -- agent skill definition

## Coding Style

- Go stdlib + cobra only. Minimal dependencies.
- stdout for data (compact plaintext or JSON). stderr for errors/progress.
- `--json` flag for structured output on all commands.
- Conventional commits: `feat(scope):`, `fix(scope):`, `chore:`.

## API

HN Algolia API at `https://hn.algolia.com/api/v1/`. Free, no auth, no rate limits.
