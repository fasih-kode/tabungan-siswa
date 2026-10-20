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

No database migration was required. Transaction entry forms are owned by
FASE 21. A separate browser E2E session was not run; automated route, service,
database integration, and template tests passed.

## Audit follow-up — 2026-10-19

- POST form validation now enforces a valid URL-encoded Content-Type and the
  16 KiB body limit before CSRF parsing. Invalid encoding, multipart requests,
  and oversized bodies cannot reach account creation.
- Route regression tests cover those rejection cases with authenticated
  sessions, valid CSRF tokens, and zero creation calls; URL-encoded forms with
  a valid charset parameter still succeed.
- Template tests explicitly cover empty OPEN/SETTLED passbooks, an empty
  directory, and a readable SETTLED account without a recreation action.
- The HTTP contract now specifies body fields, query parameters, pagination,
  validation order, redirects, and error status mappings. Its status is LOCKED.
- Reverification passed: `make test-integration` and
  `GOFLAGS=-race make test-integration`, both against the guarded
  `tabungan_siswa_fase168_clean_test` database; `go vet ./...`,
  `go build ./...`, `npm run test:instant-search` (7 tests), and
  `git diff --check` also passed.

## Post-lock scope amendment — 2026-10-20

The original FASE 20 wording assigned transaction editing and cancellation to
FASE 21. The product owner approved the Roadmap v3 boundary: FASE 21 owns
deposit/withdrawal entry and transaction reads; transaction edit, cancellation,
and correction belong to FASE 22. The FASE 20 HTTP contract and FASE 21
transaction contract are aligned with this amendment. No implementation code
is changed by this documentation amendment.
