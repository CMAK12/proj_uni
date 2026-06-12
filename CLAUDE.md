# CourseHub — AI agent rules

Project context and rules for AI coding assistants (Claude Code, Cursor, etc.).
Keep these rules in sync with `docs/patterns.md`.

## What this is
A layered Go course-management system. Dependencies point inward toward
`internal/domain`. `cmd/server` is the only composition root.

## Architecture rules
- `service` depends on **interfaces** it declares (`internal/service/ports.go`),
  never on `storage`. Concrete repos are injected in `cmd/server`.
- Define interfaces where they are **consumed**, keep them 1–3 methods.
- `domain` has no business logic and imports nothing from other internal pkgs.
- Add a grading scheme → new type in `internal/grading` + register it. Do not add
  `switch` statements over strategy names elsewhere.
- Add a reaction to grading → new `progress.Observer`, subscribe in `cmd/server`.
- Build courses only through `course.Factory`; never construct `base` directly.

## Go style
- gofmt/goimports; `MixedCaps`; exported names get doc comments starting with the name.
- Wrap errors with context (`fmt.Errorf("...: %w", err)`); compare with `errors.Is/As`.
- Use sentinel errors from `internal/domain/errors.go` for expected failures.
- `ctx context.Context` is the first parameter; never store it in a struct.
- Accept interfaces, return structs. No global state or `init()` side effects.

## Testing rules
- Table-driven tests with `t.Run` subtests; name `TestФункція_Сценарій`.
- Use stdlib `testing` only (no testify). Mark helpers with `t.Helper()`.
- Cover edge cases: empty input, boundaries, error paths. Target ≥ 70% coverage.
- Run `go test ./... -cover` and `go vet ./...` before considering work done.

## Git
- Do not run git write operations; the maintainer commits manually.
