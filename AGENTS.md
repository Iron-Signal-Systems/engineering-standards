# Repository Guidelines

## Project Structure & Module Organization

This repository defines the Iron Signal Repository Assurance Standard (ISRAS) and its Go reference tooling. Normative requirements live in `standards/`; supporting adoption material belongs in `integration-guides/`. Go commands are under `cmd/`, with reusable implementation packages in `internal/`. Keep tests beside their packages as `*_test.go`. JSON contracts and examples belong in `schemas/`, validation policy and pinned tool identities in `validation/`, release metadata in `release/`, and developer entry points in `tools/`. Store design and release evidence in `docs/records/` and `docs/releases/`; generated local evidence stays under ignored `.local/` paths.

## Build, Test, and Development Commands

- `./tools/build-validator.sh` runs the complete Go test suite and builds `.local/bin/isras-validate`.
- `./.local/bin/isras-validate all` runs development validation, including formatting, vetting, tests, builds, module checks, vulnerability checks, and secret scanning.
- `./.local/bin/isras-validate all --mode commit` validates an exact clean commit before review or release work.
- `go test -count=1 ./...` runs all tests without cached results; use `go test -count=1 ./internal/projectcommand` for a focused package.
- `go vet ./...` and `gofmt -l .` provide direct static-analysis and formatting checks.

Release scripts under `tools/build-release-*.sh` enforce additional signed-source and clean-clone requirements; do not use them as ordinary development builds.

## Coding Style & Naming Conventions

Format Go with `gofmt`; use tabs as emitted by the formatter. Follow standard Go naming: short lowercase package names, exported identifiers in `PascalCase`, and descriptive test names such as `TestExecuteRejectsUnknownFinding`. Shell scripts use Bash with strict error handling and kebab-case filenames. Keep Markdown headings descriptive and JSON schemas versioned, for example `isras-project-v1.schema.json`.

## Testing Guidelines

Every behavioral change needs a deterministic regression test. Prefer injected fakes and temporary repositories over live services. Tests must preserve fail-closed behavior, bounded execution, redaction, and exact evidence identity. No numeric coverage threshold is defined; complete applicable validation is the acceptance gate.

## Commit & Pull Request Guidelines

Use short imperative subjects, commonly prefixed with `feat:`, `fix:`, `test:`, `docs:`, `release:`, or `chore(release):`. Sign commits and release tags. Target `dev` with a focused PR that explains the problem, assurance impact, verification performed, and related issue. Update governing standards, schemas, records, examples, and release notes together when their contract changes. Do not claim independent review for self-validation.
