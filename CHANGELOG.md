# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `hn front --feed` reads the `top`, `new`, `best`, `ask`, `show` and `job` lists.
- `hn read` flags `-n`, `--replies`, `--depth` and `--max-chars`, and comment IDs
  in its output. `hn read COMMENT_ID` opens a collapsed thread.

### Changed
- `hn read` caps its output by default, and says what it leaves out. `--expand`
  shows everything.

### Fixed
- `hn front` lists stories in site rank order. Algolia's `front_page` tag has
  the right stories in another order.
- `hn read` lists comments in site rank order, not oldest first.
- Exit codes now match the docs. Network failures, rate limits and server
  errors exit 8, missing items and users exit 5, and an empty `hn front` exits 3.
- `hn status --json` exits 8 when the API is down.
- `hn user` looks profiles up through the Firebase API, because Algolia answers
  500 for unknown users.

## [0.2.1] — 2026-08-08

### Fixed
- Reverted to a Homebrew formula. Casks are macOS-only, so v0.2.0 silently
  dropped `brew install` support on Linux.

## [0.2.0] — 2026-08-08

### Changed
- Distributed as a Homebrew cask instead of a formula. `brew update` migrates
  existing installs automatically.

### Added

- `hn search` -- search stories or comments with date/points filters
- `hn front` -- current front page stories
- `hn read` -- read a story with threaded comments
- `hn user` -- user profile and karma
- `hn status` -- API health check
- `--json` flag for structured output on all commands
