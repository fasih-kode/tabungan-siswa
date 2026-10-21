# FASE 24: User Administration Implementation Lock

Status: **LOCKED**
Date: 2026-10-21
Contract: [FASE 24 User & Administration Contract](fase-24-user-administration-contract.md)
Owner acceptance: 2026-10-21
Implementation commit: `e233883` (`feat(fase-24): administer user accounts`)

## Delivered scope

- Admin account directory for WALI_KELAS and SISWA, including username update,
  activation/deactivation, and password reset.
- ADMIN identities remain provisioned through the guarded bootstrap command;
  the HTTP/UI administration path cannot create an ADMIN or change account role.
- Deactivation and password reset revoke all account sessions atomically.
  Reset credentials are shown once and require a password change at next login.
- Inactive accounts are rejected during login and on authenticated requests.
- Admin may view savings for the selected year; the Admin roster query is global
  while account lookup remains scoped to the selected year. Wali roster queries
  retain their academic-year/class scope.
- An exact `GET /` route redirects visitors to `/login`.

## Verification evidence

Checks passed on 2026-10-21:

- `GOCACHE=/tmp/tabungan-siswa-gocache make test-integration`
- `GOCACHE=/tmp/tabungan-siswa-gocache GOFLAGS=-race make test-integration`
- `GOCACHE=/tmp/tabungan-siswa-gocache go vet ./...`
- `GOCACHE=/tmp/tabungan-siswa-gocache go build ./...`
- `npm run test:instant-search` (7 tests)
- `GOCACHE=/tmp/tabungan-siswa-gocache PLAYWRIGHT_BROWSERS_PATH=/tmp/tabungan-siswa-playwright-browsers npm run test:e2e` (2/2)
- `GOCACHE=/tmp/tabungan-siswa-gocache go test ./internal/http -run 'TestNewRouterPublicLoginRoute$' -count=1`
- `git diff --check`

The browser suite used the guarded disposable database `tabungan_siswa_fase168_clean_test`, schema version 12. Its fixtures were removed automatically. Migration 12 upgrade/rollback was tested on that disposable database. The local development database was separately migrated to version 12 to resolve the Admin login schema mismatch; production was not modified.

## Boundary

This lock covers the approved FASE 24 contract. It does not add session listing,
individual session revocation, role reassignment, a durable account audit trail,
or establish production readiness. Future behavior changes require a contract
amendment and relevant tests.
