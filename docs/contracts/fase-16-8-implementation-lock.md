# FASE 16.8 — Contract Enforcement Implementation

**Status:** FINAL REVIEW IN PROGRESS — DISPOSABLE ROLLBACK TESTS PASSED
**Baseline commit:** `d4c62fe`
**Migration:** `000006_fase_16_8_contract_enforcement`
**Scope:** Implement open decisions from FASE 16.7 and close financial correctness gaps assigned to FASE 16.8.

This document records the intended coverage and final-review findings. Migration metadata recovery, code verification, and disposable-database rollback tests have been completed. The phase must not be marked `LOCKED` until the final review is complete and the changes are committed.

## 1. Decision Coverage

| Decision | Enforcement | Test coverage added/retained |
|---|---|---|
| `DEC-16-01` | Existing migration `000005` exclusion constraint | Repository integration test for overlap and adjacency; already committed in `d4c62fe` |
| `DEC-16-02` | Domain range validation, bounded history creation during transfer, PostgreSQL triggers validating history insert/update and academic-year date updates | Domain and repository integration tests for out-of-year, open-ended, and academic-year-shortening cases |
| `DEC-16-03` | Service guard `HasOpenByAcademicYear` plus PostgreSQL trigger preventing CLOSED while non-SETTLED accounts exist | Service unit test and repository integration test |
| `DEC-16-05` | Service validation plus PostgreSQL trigger requiring student status `ACTIVE` when creating an account | Service and repository integration tests |
| `DEC-16-06` | Service validation plus PostgreSQL trigger requiring academic-year status `OPEN` when creating an account | Service and repository integration tests |
| `DEC-16-07` | Service checks on transfer request, approval, and rejection; PostgreSQL trigger protects direct writes | Closed-year request service test; repository trigger exercised through migration-backed operations |
| `DEC-16-08` | Partial unique index for one `PENDING` request per student and academic year | Repository integration test expects `repository.ErrConflict` on second pending request |
| `DEC-16-09` | Settlement service check plus PostgreSQL trigger preventing account reopen while year is CLOSED | Service and repository integration tests |
| `DEC-16-10` | Existing `RequireStudentScope` verifies teacher assignment and student class history for the requested academic year | Existing `TestRequireStudentScopeIntegration` and cross-year scope tests retained; no redundant implementation added |
| `DEC-16-13` | Domain trims and validates rejection reason; PostgreSQL check constraint requires a nonblank reason only for `REJECTED` | Domain behavior and repository integration test for REJECTED without reason |

`DEC-16-04` and `DEC-16-14` remain assigned to FASE 19 and FASE 21 respectively. `DEC-16-11` and `DEC-16-12` remain deferred to FASE 22. The unresolved `TeacherClassAssignment` period policy remains a decision item and is not silently folded into `DEC-16-02`.

## 2. Financial Correctness Coverage

| Finding | Implementation |
|---|---|
| `FIN-MONEY-02` | `Money` operations reject arithmetic outside the whole-number range representable by `NUMERIC(15,2)`; balance calculation propagates overflow. Transaction and settlement constructors reject amounts above the supported maximum. |
| `FIN-BALANCE-02` | Deposit validates the resulting chronological active transaction history before persistence, including retrospective deposits. |
| `FIN-EDIT-01` | Transaction edit rejects zero, negative, and out-of-range amounts before starting the transaction. |
| `FIN-CANCEL-02` | Cancellation is rejected for SETTLED accounts in service and database trigger. |
| `FIN-CANCEL-03` | Cancellation simulates the resulting active transaction history before updating the transaction; cancellation is rejected if it would make the historical balance negative or overflow. |
| `FIN-SETTLEMENT-05` | Transactions of SETTLED accounts cannot be cancelled. |

The in-memory transaction order matches repository ordering: transaction calendar date, `created_at`, then ID. Transaction-date range enforcement remains deferred to FASE 21 under `DEC-16-14`.

## 3. Migration Behavior

Migration `000006` is wrapped in a transaction. It:

1. Refuses to proceed if finite student-class-history ranges fall outside their academic year.
2. Normalizes valid legacy open-ended histories to the academic-year exclusive end date. Original ranges and timestamps are recorded in a migration backup table. Rollback restores the original `validity` and `updated_at` only when both current values still match the normalized values recorded by the migration. Backup rows remain when the history is missing or those values no longer match.
3. Adds triggers for class-history boundaries, academic-year closure, account creation/reopening, transfer-year status, and cancellation against settled accounts.
4. Adds a partial unique index for pending transfers and a rejection-reason check constraint.
5. Runs preflight checks for duplicate pending requests and inconsistent rejection-reason data before adding those constraints.

If migration preflight fails, do not delete records or manually bypass constraints. The error should be reviewed as existing data violating the locked contract. Because the migration is wrapped in `BEGIN`/`COMMIT`, a failed `psql -f` execution should roll back as a unit when the connection closes; still inspect the exact error before retrying.

## 4. Final Verification

The development database already contains the 000006 objects and its migration metadata was reconciled to version 6 (`dirty=false`) by the incident recovery script. **Do not run the 000006 up migration again against that database.** The `up.sql` file is not idempotent.

Run the application checks against the configured development/test database:

```bash
set -a
. ./.env
set +a
export DATABASE_URL="$TABUNGAN_SISWA_DATABASE_DSN"

psql "$TABUNGAN_SISWA_DATABASE_DSN" -X -P pager=off -c \
  "TABLE public.schema_migrations;"

go test ./internal/domain -count=1
go test ./internal/repository/postgres -count=1
go test ./internal/service -count=1
go test ./...
go test -race ./...
go vet ./...
go build ./...
git diff --check
```

Test migration `up` and `down` only against a **disposable database** restored to schema version 5. Verify both directions, including rollback after a class-history row is modified or deleted after `up`: the down migration must preserve the backup table when any original range cannot safely be restored. Never experiment with rollback against the working development database.

## 5. Definition of Done

- [x] Implementations and focused tests prepared for all FASE 16.8 open decisions and financial gaps.
- [x] Database invariants protect direct repository/database writes as well as service paths where applicable.
- [x] Existing cross-year Wali Kelas scope behavior retained and covered by integration tests.
- [x] Deferred work remains assigned to its owner phase or explicit decision queue.
- [x] Migration metadata reconciled to version 6 (`dirty=false`) after validating existing 000006 objects.
- [x] Focused tests, full tests, race tests, vet, build, and diff checks passed in the user's local environment before this final rollback correction.
- [x] Validate migration `up` and `down` against disposable databases, including unchanged, modified, and deleted history rows.
- [ ] Final review and commit; only then set status to `LOCKED`.
