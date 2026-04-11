# Changelog

All notable changes to this project are documented in this file.

The format is inspired by Keep a Changelog and follows Semantic Versioning.

## [Unreleased]

### Planned
- Wails and React frontend scaffolding for desktop experience.
- Expanded E2E test coverage when UI flows land.

## [0.1.0] - 2026-04-11

### Added
- Phase 1 backend scaffold with domain, service, repository, and HTTP layers.
- Unified chat command endpoint with graph and retrieval commands.
- Semantic search and graph-data API endpoints.
- Integration tests for full API flow and error envelope consistency.
- Docker Compose stack for Redis and DGraph local services.
- CI workflow for formatting, test, and build verification.
- Release workflow for tag-driven Windows and Linux binaries plus checksums.
- Project PR template and developer runbook.

### Changed
- Error responses standardized with machine-readable code values.
- Config loading now supports defaults -> .env -> environment override precedence.

### Fixed
- Validation and internal failure classification across handlers for cleaner API behavior.
