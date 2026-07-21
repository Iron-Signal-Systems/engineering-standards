# Govulncheck typed evidence projection

**Status:** WORKSTREAM A CANDIDATE — A4.2 EVIDENCE PROJECTION ONLY — NOT RELEASED OR ADOPTABLE

## Scope

This step converts the exact verified scanner identity and deterministic per-module runner results into typed evidence structures. It does not yet alter the project-command JSON schema, JSON writer, text renderer, or runtime dispatch.

## Coverage contract

Projection fails unless at least one governed module exists, result count equals governed module count, every result references one exact governed `go.mod` path, no result path is duplicated, directory and module path match the governed selection, package scope is exactly `./...`, and every governed module has one result.

Results are sorted by `go_mod_path` before evidence is produced.

## Recorded evidence

Scanner identity records executable, directory, approved command package, embedded module, exact version, build Go version, SHA-256, package scope, and effective local/off settings.

Each module records identity, timing, exit and boundary flags, environment names, bounded stdout/stderr, scanner configuration, message counts, SBOM roots/modules, advisory IDs, and module/package/symbol/unknown finding counts.

Slices are defensively cloned so later mutation of runner results cannot alter retained evidence.

## Remaining integration

The next controlled step attaches these structures to project-command evidence schema v2, synchronizes JSON schema and examples, and renders the same content in text evidence.
