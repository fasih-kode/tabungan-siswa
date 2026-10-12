# FASE 17 — HTTP Feature Foundation Implementation Lock

**Status:** LOCKED
**Contract:** `docs/contracts/fase-17-2-http-contract.md`
**Acceptance review:** `docs/contracts/fase-17-3-acceptance-review.md`
**Final review commit:** `674f2a66215cdc0ef5ec53fd2319a9af8b79f655`
**Integration-test configuration commit:** `318e1e5ade955a173e448ac2fb42324933a2ca76`
**Authentication test reference:** `21aa0d7fbc4c1ad72799c5ebdd46748bf3521a37`
**Scope:** HTTP Feature Foundation using Go `net/http`, `html/template`, HTMX as progressive enhancement, Tailwind CSS, and PostgreSQL.

This record locks the FASE 17 HTTP foundation and its applicable contracts.
It does not claim that future feature endpoints, pagination, search/filter,
general flash messages, or partial HTMX responses have been implemented.

## 1. Contract Coverage

- [x] Handler delegates use cases to services and does not access repositories directly.
- [x] Required handler and router dependencies are validated during wiring.
- [x] HTTP error mapping preserves the general and authentication-specific boundaries.
- [x] Form submission follows Post/Redirect/Get where applicable, using HTTP 303.
- [x] Login and logout redirect behavior is covered by tests.
- [x] Logout behavior for missing or invalid cookies and service failures is covered.
- [x] Page-specific view models and Go `html/template` remain the presentation baseline.
- [x] Full-page HTML remains usable without HTMX.
- [x] Query parameters, pagination, search/filter, and partial responses remain
      feature-specific and are not claimed as implemented in this foundation.
- [x] No unrelated domain or feature implementation is included in the phase
      finalization changes.

## 2. Verification Evidence

The following checks passed in the user's local environment after the contract
and acceptance-review changes were prepared and before their final commit:

- `gofmt -l` over Go files under `cmd` and `internal` returned no filenames.
- `go test ./internal/http -count=1` passed.
- `go vet ./...` passed.
- `make test-integration` passed, including the full Go test suite.
- The integration-test script verified the connected database as
  `tabungan_siswa_fase168_clean_test`.
- `git diff --check` passed.
- `git diff --cached --check` passed before the documentation commit.

The standard integration-test command is `make test-integration`.
It derives the connection configuration from `.env`, selects the approved
disposable test database, verifies `current_database()` before testing, and
then runs `RUN_INTEGRATION_TESTS=1 go test ./... -count=1`.

The existing Go database DSN guard remains unchanged and is an independent
safeguard. Integration tests must not target `tabungan_siswa_dev`.

## 3. Applicability Boundaries

The following are intentionally not claimed as implemented in FASE 17:

- Generic list endpoint query parsing, pagination, search, and filtering.
- General flash-message infrastructure.
- Partial-response endpoints and broad HTMX interactions.

These are to be specified with the applicable feature contracts and roadmap
phases. Their absence from this foundation is not treated as a failure of
FASE 17 acceptance.

## 4. Lock Decision

The applicable FASE 17 HTTP foundation criteria have been reviewed, and the
recorded final verification passed. The contract and acceptance review are
committed in `674f2a6`.

This document records the phase-lock decision. The commit containing this
document is the Git record that makes this lock decision part of repository
history. Do not amend the scope above to imply that deferred features exist.
