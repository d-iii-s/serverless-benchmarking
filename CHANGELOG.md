# Changelog

All notable changes to this project are documented in this file.

## [3.0.0] - 2026-04-14

### Added
- `probe-bodies` command for OpenAPI link-aware, Schemathesis-based chain generation.
- `harness` command for stage-oriented workload execution with first-response metrics and container stats.
- Flow DSL validation with JSON Schema and weighted transition support.
- `--max-probe-target`, `--service-mount-path`, and `--debug-non2xx` flags.
- Release notes artifact file `RELEASE_NOTES.md`.

### Changed
- README rewritten and expanded with architecture/lifecycle diagrams, DooD topology, and thesis-oriented framing.
- CI workflows updated to test current packages and build release artifacts.
- Docker usage standardized on versioned image tag examples (`aape2k/slsbench:v3.0.0`).

### Removed
- Legacy command paths and related dead code (`enrich`, `scenario`, `walker` lineage).
- Unused dependencies and stale utility/model layers.
- Tracked local binaries and Python bytecode artifacts from version control.
