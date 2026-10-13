# FASE 18: Academic Year HTTP Vertical Slice

Status: IMPLEMENTED AND VERIFIED

## Scope

This phase exposes the existing Academic Year service through server-rendered HTML. It does not add business rules to the HTTP layer and does not replace the service/domain/database contracts.

## Routes

| Method | Path | Purpose | Service operation |
|---|---|---|---|
| GET | `/academic-years` | List academic years | `List` |
| POST | `/academic-years` | Create an academic year | `Create` |
| GET | `/academic-years/{id}` | Show one academic year | `Get` |
| POST | `/academic-years/{id}/start-closing` | Move OPEN to CLOSING | `StartClosing` |
| POST | `/academic-years/{id}/close` | Move CLOSING to CLOSED | `Close` |
| POST | `/academic-years/{id}/reopen` | Move CLOSED to OPEN | `Reopen` |

All routes require authentication. Unsafe methods require CSRF validation. Successful POST requests use HTTP 303 redirects.

## Authorization

The actor is read from the authenticated request context populated by authentication middleware. The handler does not accept user ID or role from form fields or query parameters. The service remains authoritative for authorization:

- Create and lifecycle transitions: Admin only.
- Read/list: Admin, Wali Kelas, and Siswa.

## Input and pagination

- Create fields: `name`, `start_date`, `end_date`.
- Dates use ISO `YYYY-MM-DD` form values.
- `page` defaults to 1 and must be positive.
- `limit` defaults to 20 and must be between 1 and 100.
- Repository/service pagination uses `Limit` and `Offset`; list order is defined by the repository.
- Invalid page or limit values return HTTP 400.

## Error handling

Known service/domain/repository errors use the shared `httperror` mapping. Form validation errors are rendered as safe, user-facing messages. Internal errors are not exposed to the page. Unknown errors remain HTTP 500.

## Presentation

Pages use `html/template` and the shared `base` layout with the `page-content` template block. View models contain presentation values only. Create and transition actions are standard HTML forms with CSRF tokens. HTMX behavior is not introduced by this phase.

## Verification requirements

- Unit tests for route wiring, form validation, pagination (including lookahead boundary behavior), role context, redirects, error mapping, and rendering.
- Full `go test ./...` and `go vet ./...`.
- Integration tests must use the approved isolated test database via `make test-integration` / `scripts/test-integration.sh`. Never point integration tests at the development database.
- Verification was run with the required project Go toolchain. `go test ./internal/http -count=1`, `go vet ./...`, and `make test-integration` all passed. The integration target verified the isolated database `tabungan_siswa_fase168_clean_test`.
