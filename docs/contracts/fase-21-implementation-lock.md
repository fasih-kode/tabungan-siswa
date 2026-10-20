# FASE 21: Transaction Engine and HTTP Implementation Lock

Status: LOCKED  
Date: 2026-10-20  
Contract: [FASE 21 Transaction HTTP Contract](fase-21-transaction-http-contract.md)  
Implementation commit: `fe21e78` (`feat: deliver FASE 21 transactions and year-close gates`)

## Delivered scope

- Admin and scoped Wali Kelas can record deposits and withdrawals for eligible
  OPEN savings accounts in OPEN academic years.
- Transaction date, amount, chronological balance, authorization, audit
  provenance, account/year state, and concurrent writes follow the locked
  contract.
- Authorized users can view paginated transaction history and transaction
  details with effective-date running balances.
- HTTP routes, CSRF and form validation, redirects, error messages, and
  templates implement the reviewed contract.
- PostgreSQL locking and integration coverage include year transition and
  settlement interleavings, same-account serialization, and independent
  accounts.
- The separately approved lifecycle prerequisite is enforced: creating a new
  academic year requires all existing years to be CLOSED, with a database
  trigger guarding direct inserts. Close failures for unsettled accounts or
  pending class transfers are explained on the year detail page.

## Verification evidence

The following checks passed on 2026-10-20:

- `make test-integration`
- `GOFLAGS=-race make test-integration`
- `go vet ./...`
- `go build ./...`
- `npm run test:instant-search` (7 tests)
- `git diff --check`

The Go integration and race suites used the guarded disposable PostgreSQL test
database. HTTP, service, template, PostgreSQL repository, and concurrency
coverage are included in the implementation commit. Browser E2E was not run.

## Boundary

This lock covers FASE 21 only. Transaction correction/edit/cancel remains in
FASE 22. Settlement, account reopening, and year-end close workflows remain
in FASE 23. No settlement or correction behavior is implied by this lock.
