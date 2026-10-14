# Changelog

All notable changes are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org).

## [Unreleased]

### Added

- The plan, architecture, UX and research notes.
- Project scaffolding: licence, community files, CI, release configuration and decision records.
- Host runtime: one holder process per session, with a PTY, an 8 MiB output ring with byte offsets, a manifest, and status from output signals. Each session runs as a systemd user service with a memory cap.
- Host helper commands: `bridge`, `new`, `ls`, `rm`, `doctor`, `version`.
- Wire protocol with a versioned hello, requests, attach streams and uploads.
