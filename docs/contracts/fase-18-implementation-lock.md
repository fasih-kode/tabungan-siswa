# FASE 18: Academic Management Implementation Lock

Status: LOCKED — AMENDED
Original lock date: 2026-10-14
Authorization amendment: 2026-10-19

FASE 18 is complete as a BE + FE vertical slice for academic year lifecycle,
class management, teacher-class assignment, and class-transfer requests.

## Contract checkpoints

- Academic year: [FASE 18 Academic Year HTTP Contract](fase-18-academic-year-http-contract.md)
- Class management: [FASE 18 Class HTTP Contract](fase-18-class-http-contract.md)
- Teacher assignment: [FASE 18 Teacher Assignment HTTP Contract](fase-18-teacher-assignment-http-contract.md)
- Class transfer: [FASE 18 Class Transfer HTTP Contract](fase-18-class-transfer-http-contract.md)

## Lifecycle decision

DEC-18-TR-01 keeps an academic year `OPEN` while any class-transfer request
remains `PENDING`. Both `StartClosing` and `Close` return the existing
`ErrInvalidTransition` result without modifying the year or request. The
`StartClosing` service locks the academic-year row before checking requests,
serializing against transfer writes that take a share lock on that row. The
`Close` check remains as a defensive guard for existing `CLOSING` records.

## Verification

- Focused class-transfer HTTP and academic-year template tests passed.
- `make test-integration` passed all packages. Its database guard verified
  `tabungan_siswa_fase168_clean_test` before running tests.
- `go test -race ./... -count=1` passed. The race-test wrapper independently
  verified the same disposable database before running tests.
- `go vet ./...` passed.
- `go build ./...` passed.
- `git diff --check` passed.
- PostgreSQL integration tests cover transfer request, approval, rejection,
  pending-transfer start/close guards, and serialization of an in-flight
  request against `StartClosing`.
- HTTP tests cover authentication, CSRF, Admin-only pending access, actor
  propagation, request/approval/rejection forms, redirects, pagination, and
  safe error handling.
- Template rendering tests verify the Admin and Wali Kelas academic-year page
  is not replaced by the dashboard content.

No migration or recovery script was run or changed for the original FASE 18
implementation.

## Post-lock amendment — class-transfer requester role

The product owner clarified on 2026-10-19 that Wali Kelas submits class
transfer requests and Admin reviews them. The class-transfer HTTP contract and
implementation were amended accordingly: request controls are Wali-only, the
request handler denies Admin, and the service rejects Admin for both
`Request` and `RequestOptions`. Admin review and read permissions are
unchanged. `make test-integration`, `go test -race ./... -count=1`,
`go vet ./...`, `go build ./...`, and `git diff --check` passed after this
amendment. See the class-transfer contract for the detailed decision and
verification record.

## Review result

The implementation reuses the existing class-transfer service and its domain
rules. HTTP handlers do not read repositories or replace service authorization
and lifecycle decisions. All phase verification is complete; this checkpoint
locks the FASE 18 contracts and implementation together.

## Post-lock amendment — next-year creation prerequisite

Product owner decision, 2026-10-20: a new academic year may be created only
after all existing years are `CLOSED`; the first year is exempt. `Create`
locks the existing year rows in deterministic order and checks their status
inside its transaction. Migration `000010` adds a PostgreSQL insert trigger as
a backstop for direct writes. The existing close guard ensures all associated
savings accounts are `SETTLED` before a year reaches `CLOSED`.

PostgreSQL integration tests verify OPEN/CLOSING rejection, CLOSED acceptance,
direct-insert trigger enforcement, and serialization of lifecycle updates
against the `ListForUpdate` lock. Race-enabled integration tests, build, vet,
frontend tests, and `git diff --check` passed after the amendment.
