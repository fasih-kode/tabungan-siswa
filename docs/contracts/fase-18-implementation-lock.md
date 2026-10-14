# FASE 18: Academic Management Implementation Lock

Status: LOCKED
Date: 2026-10-14

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

No migration or recovery script was run or changed for this phase.

## Review result

The implementation reuses the existing class-transfer service and its domain
rules. HTTP handlers do not read repositories or replace service authorization
and lifecycle decisions. All phase verification is complete; this checkpoint
locks the FASE 18 contracts and implementation together.
