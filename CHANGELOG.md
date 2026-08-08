# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
