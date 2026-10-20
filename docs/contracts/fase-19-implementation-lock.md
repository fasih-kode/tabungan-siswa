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

## Post-lock lifecycle amendment — academic-year promotion order

Product owner decision, 2026-10-20: the existing FASE 19 promotion workflow is
amended to require the source academic year to be `CLOSED`. FASE 18's academic
year creation contract now requires every existing year to be `CLOSED` before
creating the next year (the first year is exempt). Since closure already
requires all associated savings accounts to be `SETTLED`, the operational
order is settlement → close old year → create new year → create target classes
→ promote/graduate students. The service, database trigger, and integration
tests must enforce this order.

- Apply migrations through `000010_academic_year_create_requires_closed_previous` before
  running this application version.
- Student account creation remains separate from student import/creation.
- Transfer request submission remains Wali-only; Admin handles review.
- This lock records automated and integration verification. No claim of a
  separate end-to-end browser session is made.
