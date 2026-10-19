# FASE 19: Student Management Implementation Lock

Status: LOCKED
Date: 2026-10-19

FASE 19 is complete as a student-management vertical slice covering student
master records, class placement/history, leaving/reactivation, scoped student
views, and spreadsheet import.

## Contract checkpoint

- [FASE 19 Student Management HTTP Contract](fase-19-student-http-contract.md)
- [Class and student placement workflow](student-class-placement-workflow.md)
- [Student promotion workflow](../student-promotion-workflow.md) records the
  adjacent bulk-promotion and graduation behavior implemented with this slice.

The class-transfer requester-role clarification is recorded separately as a
post-lock amendment to [FASE 18](fase-18-implementation-lock.md): only Wali
Kelas may submit; Admin reviews and decides.

## Verification

- `make test-integration` passed all packages against the guarded
  `tabungan_siswa_fase168_clean_test` database.
- `go test -race ./... -count=1` passed against that disposable database.
- `go vet ./...` passed.
- `go build ./...` passed.
- `npm run test:instant-search` passed all seven tests.
- `git diff --check` passed.
- Tests cover student lifecycle and class-history transactions, Wali/Admin/Siswa
  authorization, import validation and rollback, scoped lists, and HTTP/template
  behavior.

## Operational notes

- Apply migrations through `000009_user_password_change_required` before
  running this application version.
- Student account creation remains separate from student import/creation.
- Transfer request submission remains Wali-only; Admin handles review.
- This lock records automated and integration verification. No claim of a
  separate end-to-end browser session is made.
