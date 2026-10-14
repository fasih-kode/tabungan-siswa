# FASE 18: Class Management HTTP Contract

Status: VERIFIED AND COMMITTED

Implementation commit: `cb890c7` (`feat(class): add class management HTTP pages`)
Remote: `origin/main` was updated successfully.

Verification completed in the project environment on 2026-10-13:
- `go test ./internal/http -run 'TestClassRoutes' -count=1`: passed.
- `go vet ./...`: passed.
- `git diff --check`: passed.
- `make test-integration`: passed all packages.
- Integration test target verified database `tabungan_siswa_fase168_clean_test`.

## Scope

This slice exposes the existing class service through `html/template` pages. It does not replace or duplicate class business rules in the HTTP layer.

## Routes

- `GET /classes`: authenticated class list; requires `academic_year_id` to show class rows.
- `POST /classes`: create class; Admin only by service authorization.
- `GET /classes/{id}`: class detail/edit form; Admin only by service authorization.
- `POST /classes/{id}`: update class; Admin only by service authorization.

All routes require authentication. Unsafe methods pass through CSRF middleware and a 16 KiB form-body limit. Successful POST requests redirect with HTTP 303.

## Access and scope

- Admin: may list classes and create/update/detail classes.
- Wali Kelas: list is scoped by the class service to assignments in the selected academic year.
- Siswa: list is scoped by the class service to class history in the selected academic year.
- Detail/create/update authorization remains enforced by the existing service. The HTTP layer does not treat hiding links as authorization.

## Input and pagination

- `academic_year_id` is required to load the class list and must be a UUID.
- `page` defaults to 1; `limit` defaults to 20 and must be in 1..100.
- Class name is required; level must be an integer from 1 through 12.
- Duplicate name/level and domain validation errors are rendered as safe user-facing messages.

## Presentation

The class page uses `html/template`, includes an academic-year selector, a class table, an empty state, Admin create/edit forms, and validation feedback. No HTMX behavior is introduced in this slice.

## Acceptance-test coverage

The route authorization tests cover authenticated list access for Admin, Wali Kelas, and Siswa; non-Admin denial for create/detail/update; Admin create redirect; CSRF rejection before service invocation; and unauthenticated rejection before service invocation.

Handler tests cover class list rendering, service input scoping, create/update redirects, invalid class level, bounded pagination and lookahead boundaries, invalid pagination, service errors, and retention of submitted values after validation/service errors.

## Verification status

The implementation was verified in the project environment and committed as `cb890c7`. The follow-up contract/test hardening is a separate change and must pass the relevant Go tests, `go vet ./...`, `git diff --check`, and `make test-integration` before it is committed.
