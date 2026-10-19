# FASE 20 — Savings Account and Passbook HTTP Contract

Status: VERIFIED — READY TO LOCK
Scope: server-rendered savings-cycle directory, account detail, and passbook.

## 1. Roadmap scope

This vertical slice exposes savings accounts by student and academic year,
account balance, and deterministic transaction history. Deposit and withdrawal
entry/edit forms remain in FASE 21; FASE 20 may display an empty passbook.

## 2. Business rules inherited from the domain and service

- A savings cycle belongs to one `(student_id, academic_year_id)` pair. The
  database unique constraint permits at most one account row for that pair,
  including settled accounts.
- Account creation is allowed to Admin and Wali Kelas, for an ACTIVE student
  and OPEN academic year. Wali Kelas must have that student in their assigned
  class scope for the selected year. Duplicate creation follows the existing
  conflict mapping.
- Admin may read any account. Wali Kelas may read accounts for students in
  their year/class scope. Siswa may read only their own linked student's
  account. No role may infer access from a submitted student or account ID.
- Account balances use ACTIVE transactions only. Passbook entries are ordered
  by transaction date, creation timestamp, then transaction UUID, ascending;
  each entry shows the resulting balance. This provides a stable tie-break for
  same-day transactions. Cancelled transactions do not affect the balance.
- A settled account remains readable as historical data but cannot accept new
  transactions; write workflow belongs to FASE 21 and settlement lifecycle.

## 3. HTTP surface

| Method | Path | Purpose |
|---|---|---|
| GET | `/savings` | Account cycles for a selected academic year and scoped students |
| POST | `/savings` | Create a cycle for an eligible student/year |
| GET | `/savings/{id}` | Account detail, current balance, and passbook |

The year selector defaults to the OPEN academic year when present. Historical
years remain selectable for reading. Admin may search all student records.
Wali Kelas sees only students in assigned classes for the selected year and may
filter to one assigned class. Siswa sees only their own account cycle for the
selected year.
If a cycle does not exist, the page explains that state; only Admin/Wali may
create it, and only when the student is ACTIVE and the year is OPEN.

All routes require authentication. POST requires CSRF, a bounded
`application/x-www-form-urlencoded` body, actor identity from request context,
and a 303 redirect after success. Services remain authoritative for role,
scope, lifecycle, uniqueness, and balance rules.

## 4. Verification before lock

- Unit/integration coverage for account uniqueness, year isolation, role/scope
  access, and deterministic running balances.
- Authenticated HTTP/CSRF tests for list, create, and detail routes, including
  cross-student denial and Wali class scope.
- Template tests for empty, open, settled, and passbook states.
- `make test-integration`, race tests, `go vet ./...`, `go build ./...`,
  `npm run test:instant-search`, and `git diff --check` pass.
- Commit and implementation-lock record.

## 5. Final verification — 2026-10-19

- `make test-integration` passed all packages against the guarded
  `tabungan_siswa_fase168_clean_test` database.
- `go test -race ./... -count=1` passed against that disposable database.
- `go vet ./...`, `go build ./...`, `npm run test:instant-search`, and
  `git diff --check` passed.
- Tests cover account/year balance isolation, unique student-year cycles,
  passbook ordering and running balances, cross-student denial, Wali scope,
  authenticated routes, CSRF, create redirects, and template states.
- A separate browser E2E session was not run; route, service, integration, and
  template behavior were verified by automated tests.

The implementation lock is recorded in
[`fase-20-implementation-lock.md`](fase-20-implementation-lock.md).
