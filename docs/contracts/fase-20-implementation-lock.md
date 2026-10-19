# FASE 20: Savings Account and Passbook Implementation Lock

Status: LOCKED
Date: 2026-10-19

FASE 20 is complete as a savings-cycle and passbook vertical slice. Admin and
scoped Wali Kelas can create cycles for eligible students in an open year;
Admin, scoped Wali Kelas, and each student's own account can read the balance
and passbook according to the authorization contract.

## Contract checkpoint

- [FASE 20 Savings HTTP Contract](fase-20-savings-http-contract.md)
- The existing database uniqueness constraint enforces one account row per
  student and academic year.
- The existing FASE 18/19 lifecycle contracts govern academic years, student
  status, and class scope.

## Verification

- `make test-integration` passed all Go packages against the guarded
  `tabungan_siswa_fase168_clean_test` database.
- `go test -race ./... -count=1` passed against the same disposable database.
- `go vet ./...` and `go build ./...` passed.
- `npm run test:instant-search` passed all seven tests.
- `git diff --check` passed.
- PostgreSQL integration tests verify independent account balances and the
  one-account-per-student-year constraint. Unit/HTTP/template tests cover
  deterministic running balances, role scope, CSRF, account creation, and
  empty/populated passbook views.

No database migration was required. Transaction entry, editing, and cancellation
forms remain in FASE 21. A separate browser E2E session was not run; automated
route, service, database integration, and template tests passed.
