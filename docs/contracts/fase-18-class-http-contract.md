# FASE 18: Class Management HTTP Contract

Status: IMPLEMENTED IN SOURCE ARCHIVE; VERIFICATION PENDING

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

## Verification status

Source-level implementation and tests are included. The required Go tests, `go vet ./...`, and `make test-integration` must pass in the project environment before this slice is LOCKED. Do not treat source creation alone as a lock.
