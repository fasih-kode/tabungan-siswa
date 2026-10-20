# FASE 21 — Transaction Engine and HTTP Contract

Contract status: LOCKED  
Implementation status: IMPLEMENTED; VERIFICATION PASSED; LOCKED
Roadmap: Roadmap Resmi Tabungan Siswa v3, FASE 21  
Scope: deposit, withdrawal, transaction history/detail, and transaction forms.

This document defines the business, authorization, persistence, and HTTP
contract before implementation. It does not authorize code or migration
changes. Any change to a locked decision below requires an explicit contract
amendment before implementation.

Product owner approval: 2026-10-20. The three decisions in §11 were approved:
the scope and role matrix, effective-date chronological balance semantics for
backdated transactions, and FASE 22 ownership of transaction corrections.

## 1. Goal and boundary

Deliver the first usable financial transaction vertical slice for each savings
account: an authorized staff member can record deposits and withdrawals, and
authorized users can inspect transactions and their effect on the passbook
balance. Every accepted write must preserve a non-negative chronological
balance and must be atomic with its audit metadata.

Included:

- Record deposit and withdrawal.
- Validate account/year lifecycle, transaction date, amount, role, and student
  scope.
- Account transaction list, transaction detail, and deposit/withdrawal forms.
- Database transaction and account-row serialization for balance-changing
  writes.
- Create audit metadata, without exposing the audit viewer in this phase.
- PostgreSQL integration coverage for integrity and concurrency.

Excluded from FASE 21:

- Edit, cancel, or hard-delete transaction. These belong to FASE 22, consistent
  with Roadmap v3 and the aligned FASE 20 contract.
- Settlement, account reopen, academic-year closing, and carry-over. These
  belong to FASE 23.
- Audit viewer and general audit administration. The create event is persisted
  here; audit UI and full mutation coverage belong to FASE 22.
- Bulk import, recurring transactions, payment integrations, receipts, and
  balance snapshots/materialized balances.
- HTMX-specific behavior. Full HTML form submission is the baseline; broad
  enhancement remains FASE 26.

## 2. Locked decisions and inherited contracts

The following are binding inputs to this contract:

- DEC-16-14: transactions may be backdated, but their transaction date must be
  inside the related academic year's period.
- The same date-boundary invariant applies to the existing transaction Edit
  service operation, although the correction UI and correction workflow remain
  owned by FASE 22. Cancel does not alter a transaction date.
- Date interval is `academic_year.start_date <= transaction_date <
  academic_year.end_date`; the start is inclusive and the end is exclusive.
- A transaction belongs to the academic year reached through its savings
  account. A client cannot choose a separate year for a transaction.
- Only ADMIN and WALI_KELAS may create financial transactions. SISWA is
  read-only and can access only their own linked student/account.
- WALI_KELAS access is limited to the assigned class/student scope in the
  account's academic year. ADMIN has system-wide account scope.
- Savings account balance is derived from ACTIVE transactions. CANCELLED
  transactions do not affect balance; FASE 21 does not create cancelled rows.
- Transaction ordering is transaction date, creation timestamp, then UUID,
  ascending. This same order defines running balance and passbook order.
- Money is an integer amount in rupiah (no fractional amount); transaction
  amount must be greater than zero.
- Account mutations and their audit records share one database transaction.
- FASE 20's account/passbook visibility rules continue to apply.

## 3. Business invariants

### 3.1 Common write rules

For either operation, all conditions below must hold at the write's serialized
decision point:

1. Actor is authenticated, valid, and has role ADMIN or WALI_KELAS.
2. Savings account exists and is OPEN.
3. Its academic year exists and is OPEN. A CLOSING or CLOSED year rejects new
   transactions, even when the supplied transaction date is in range.
4. Account's student is eligible under the existing savings-cycle lifecycle;
   the transaction cannot change student or academic-year ownership.
5. WALI_KELAS is assigned to the student's class scope for that academic year.
6. Transaction date is a calendar date in the account's academic-year interval:
   `start_date <= date < end_date`.
7. Amount is a positive whole number of rupiah representable by the domain and
   database types. Invalid, fractional, zero, negative, or overflowing input is
   rejected before persistence.
8. The ordered balance after inserting the transaction is non-negative at
   every point in the account's chronological ledger, not merely at the final
   balance.
9. Transaction row and its CREATE audit event commit together or neither is
   visible.

The year and account checks are repeated by the service under the relevant
database transaction/locks; UI state is never authoritative.

### 3.2 Backdated transaction semantics

The transaction date is the effective date for balance ordering. Backdating is
allowed within the academic-year interval. Since it can alter balances after
that date, the service recalculates the complete ordered ACTIVE ledger with the
candidate transaction included.

- A backdated deposit is accepted only if all chronological balances remain
  non-negative (a positive deposit cannot invalidate an already-valid ledger).
- A withdrawal is accepted only if the balance never becomes negative at any
  chronological point, including later-dated transactions after the proposed
  withdrawal.
- Validation is against the resulting ledger, not only today's/final balance.
- Existing ledger inconsistency blocks a new write; it is not silently repaired
  in FASE 21.

This gives operational meaning to DEC-16-14 and the locked invariant that
balances never become negative. No extra restriction such as `transaction_date
<= today` is introduced by this contract.

### 3.3 Deposit

- Creates one DEPOSIT transaction with ACTIVE status.
- Amount increases balance from its effective date onward.
- `created_by` is the authenticated actor, never a submitted user ID.
- Duplicate submissions may create duplicate deposits because FASE 21 does not
  define an idempotency key. The UI must prevent accidental double submission
  while pending; exactly-once delivery is not promised.

### 3.4 Withdrawal

- Creates one WITHDRAWAL transaction with ACTIVE status.
- Amount decreases balance from its effective date onward.
- The resulting chronological ledger must remain non-negative.
- Concurrent withdrawals against one account must serialize so that both
  cannot rely on the same stale balance. Concurrent writes to different
  accounts need not block one another.

### 3.5 Read behavior

- Transaction list and detail are read-only in this phase.
- List includes ACTIVE transactions only; no FASE 21 operation creates
  CANCELLED transactions. The balance column is the running balance after each
  transaction in the canonical ordering.
- Account passbook and transaction list must use the same ordering and balance
  calculation. Pagination must not reset the running balance at a page
  boundary; the displayed balance for each row is computed from all preceding
  ACTIVE transactions in the account ledger.
- Detail shows transaction type, amount, effective date, recorded timestamp,
  transaction ID, and the owning account/student/year. It does not expose
  secrets or internal database details.
- `created_by` may be shown as the staff user's display name only if it can be
  loaded without widening the account authorization scope. Otherwise it may be
  omitted; authorization never derives from the creator.

## 4. Authorization matrix

| Action | ADMIN | WALI_KELAS | SISWA |
|---|---|---|---|
| List transactions | Any account | Account/student in assigned year/class scope | Own linked student's account only |
| Read transaction detail | Any transaction | Transaction whose account is in scope | Own linked student's transaction only |
| Record deposit | Any OPEN eligible account/year | OPEN eligible account/year in scope | Deny |
| Record withdrawal | Any OPEN eligible account/year | OPEN eligible account/year in scope | Deny |
| Edit/cancel | FASE 22 | FASE 22, scoped | Deny |

Authorization is always derived through transaction → savings account → student
and academic year. Never authorize from a submitted transaction ID, account ID,
student ID, or `created_by` alone. For an unknown resource, preserve the
existing service/HTTP not-found semantics; do not leak cross-scope existence.

## 5. Persistence and concurrency contract

1. Each write executes within one PostgreSQL transaction.
2. Use this lock order for deposit and withdrawal. The existing Edit service
   must use the same year-before-account order when validating its replacement
   effective date; Edit workflow remains FASE 22.

   1. Read the account without a row lock only to discover its immutable
      `academic_year_id`.
   2. Lock that academic-year row `FOR SHARE` and verify that it is OPEN and
      that the effective date is within its bounds.
   3. Lock the savings-account row `FOR UPDATE`; verify that it still belongs
      to the discovered year and remains OPEN.
   4. Keep both locks through ledger validation, transaction insertion, audit
      insertion, and commit.

   The year lock serializes a transaction with `StartClosing`/`Close`, which
   take `FOR UPDATE` on the year. The account lock serializes it with other
   financial mutations and settlement. A request that obtains the shared year
   lock first completes while the year is OPEN before closing can advance; a
   request that waits behind closing re-reads the committed year state and is
   rejected unless it remains OPEN.
3. Read all ACTIVE transactions needed to validate the complete chronological
   ledger; do not validate from a paginated subset or cached balance.
4. Insert transaction and audit event atomically. On audit failure, transaction
   insertion rolls back. On commit failure, report failure and do not claim
   success.
5. Audit event identifies actor, action CREATE, entity type/ID, timestamp, and
   after-state sufficient to identify type, amount, date, account, and creator.
   No password, password hash, session token/secret, or CSRF token is recorded.
6. Database constraints remain a final guard for positive integral amount,
   valid transaction type/status, and foreign keys. Service validation remains
   responsible for academic-year date range and chronological balance.
7. The contract does not mandate a new migration. Inspect existing constraints
   and repository locking before deciding whether schema changes are necessary.
8. The lock order in item 2 is normative for FASE 21. Do not acquire the
   account lock and then request a year row lock in transaction writes.
9. FASE 21 does not change settlement or year lifecycle lock order. Their
   current paths are documented in §12; integration tests must verify their
   interaction with the normative transaction order and detect deadlocks.

## 6. HTTP surface

The route names below are the intended contract. They may be adjusted only
before lock if the existing router conventions require it, without changing
semantics.

| Method | Path | Purpose |
|---|---|---|
| GET | `/savings/{account_id}/transactions` | Paginated account transaction list |
| GET | `/transactions/{transaction_id}` | Transaction detail |
| POST | `/savings/{account_id}/transactions/deposits` | Record deposit |
| POST | `/savings/{account_id}/transactions/withdrawals` | Record withdrawal |

The savings account detail/passbook page links to the list and displays
deposit/withdrawal forms only to ADMIN/WALI_KELAS when the account and year are
OPEN. A hidden/disabled form is presentation only; services enforce all rules.
GET routes support HEAD. Unsupported methods return 405 with `Allow`.

### 6.1 Request format

POST accepts only `application/x-www-form-urlencoded`; valid media-type
parameters such as `charset=UTF-8` are allowed. Limit the full body to 16 KiB.
Parse and reject malformed/oversized/unsupported form bodies before CSRF
validation, matching the FASE 20 savings form contract.

Deposit and withdrawal body fields:

| Field | Requirement |
|---|---|
| `amount` | Required positive whole-rupiah decimal integer, no grouping separators or decimal fraction |
| `transaction_date` | Required ISO calendar date `YYYY-MM-DD`; must satisfy the academic-year interval |
| `csrf_token` | Token matching CSRF cookie; shared CSRF header may supply it instead |

`account_id` comes only from the route. Actor comes only from authenticated
context. Ignore/reject conflicting IDs or fields supplied in body/query; they
must not redirect a transaction to another account or actor. Do not accept
client-supplied status, type, created_by, academic year, balance, or audit data.

List query parameters:

| Parameter | Behavior |
|---|---|
| `page` | Positive integer, default 1; overflow/invalid values return 400 |
| `limit` | 1–100 accepted for compatibility; effective page size remains the shared fixed size of 10 |

The transaction list is scoped to the route account. Pagination links preserve
filters and page balances remain ledger-correct as defined in §3.5. Detail
requires a non-zero transaction UUID.

### 6.2 Validation and response ordering

For all endpoints, authentication runs before handler logic. For POST:

1. Method and authenticated actor/role gate.
2. Content-Type, size limit, and form encoding validation.
3. CSRF validation.
4. Route UUID and form field syntax/range validation.
5. Service-level resource lookup, authorization, lifecycle, academic-year date
   boundary, ledger validity, and mutation.
6. Atomic commit; redirect only after successful commit.

Service/database checks remain authoritative even if the handler rejects
obvious malformed input first. Resource existence may be reported before
scope denial according to existing service semantics.

| Condition | Response |
|---|---|
| List/detail success | 200 HTML |
| Successful POST | 303 See Other to account passbook/list with a non-authoritative success notice |
| Missing/invalid authentication | 401 |
| Unsupported method | 405 and `Allow` |
| Malformed body, UUID, date syntax, amount syntax/range, page, or limit | 400 |
| Missing/invalid CSRF on otherwise valid POST | 403 |
| Forbidden role or out-of-scope student/account | 403 |
| Account/transaction/year not found | 404 |
| Settled account, non-OPEN year, out-of-period date, negative chronological balance, or invalid lifecycle transition | 409 |
| Unexpected internal/database failure | 500 with generic public text |

No response may expose SQL, constraint names, raw database messages, stack
traces, or internal identifiers beyond resource IDs already in the request.
GET pages use HTML; this phase does not add JSON endpoints or HTMX-only partial
responses.

## 7. Presentation contract

- Account detail remains the canonical passbook context and shows current
  balance, account/year/student identity, and links to transaction history.
- Staff sees deposit and withdrawal forms only for an OPEN account in an OPEN
  year; otherwise the page states why writes are unavailable.
- Siswa sees read-only balance and transaction history for their own account.
- Forms label amount as whole rupiah and date as transaction/effective date.
- Validation errors preserve safe submitted amount/date values where practical;
  they never echo untrusted HTML.
- Success notices are informational only and cannot trigger or repeat a write.
- Normal form POST/redirect/GET works without JavaScript. No JS/HTMX behavior
  may bypass server validation or CSRF.

## 8. Error semantics

Use established domain/service errors and HTTP error mapping where their
semantics match. Distinguish malformed request (400) from validly formed but
business-invalid transaction (409). Do not map every domain error to 400.
Unexpected failures return generic text and are logged through established
logging practices without credentials, session secrets, or full CSRF values.

## 9. Verification and exit criteria

FASE 21 may be locked only when all of the following are demonstrated:

- Unit coverage for amount/date parsing, date interval boundaries, ordering,
  chronological balance validation, and error mapping.
- Service tests for ADMIN/WALI_KELAS/SISWA role behavior, Wali scope, OPEN
  account/year requirements, actor provenance, and atomic audit behavior.
- PostgreSQL integration tests for deposits, withdrawals, failed audit rollback,
  year-boundary dates, backdated ledger outcomes, and independent accounts.
- A real concurrency test proves two withdrawals on one account cannot both
  consume the same balance; a concurrent write on another account is not
  incorrectly coupled to the first account's lock.
- HTTP tests cover auth, CSRF, body size/content type, malformed input,
  redirects, scope denial, status mappings, and no mutation on rejected input.
- Template tests cover staff writable/read-only states, student read-only state,
  empty and populated passbooks, and error/success notices.
- `make test-integration` and the established race/integration command pass
  against the guarded disposable PostgreSQL database; `go vet ./...`,
  `go build ./...`, `npm run test:instant-search`, and `git diff --check` pass.
- Contract review confirms FASE 20 boundary amendment, implementation review,
  commit, and phase lock record.
- Any required schema change has a reviewed migration and safe upgrade/rollback
  evidence. A migration is not presumed necessary.
- No separate browser E2E claim is made unless a browser session is actually
  run and recorded.

## 10. Explicit non-goals and deferred decisions

- Transaction edit/cancel policies, including preserving valid historical
  balances after corrections: FASE 22 contract.
- Settlement's behavior with backdated transactions and settlement snapshots:
  coordinate with FASE 23; FASE 21 cannot mutate a SETTLED account.
- Account/year reopen workflows and approval: FASE 23.
- Idempotency guarantees for retried POSTs: not specified in current roadmap;
  requires a separate decision if exactly-once semantics are needed.
- Receipt generation, export, and notification: not in FASE 21.
- Audit viewer and general audit retention policy: FASE 22 or operational
  contract, while required CREATE audit records are written in FASE 21.

## 11. Contract review checklist

- [x] Product owner confirms the scope and role matrix.
- [x] Product owner confirms backdated ledger semantics in §3.2 are the
      intended interpretation of DEC-16-14 and the non-negative balance rule.
- [x] Product owner confirms correction/edit/cancel are owned by FASE 22.
- [x] HTTP route and field contract reviewed as part of the approved contract.
- [x] Contract status set to LOCKED before implementation.
- [x] Implementation follows the approved transaction lock order and date
      invariants.
- [x] PostgreSQL integration tests cover account/year lock interleavings,
      settlement races, chronological balance validation, and date boundaries.
- [x] Build, vet, frontend tests, and race-enabled PostgreSQL integration suite
      pass.
- [x] Phase lock record and implementation commit created after owner review.

Implementation must follow the normative lock order in §5 and satisfy the
concurrency verification in §12. This is an implementation constraint, not an
unresolved product decision.

## 12. Lock-order inspection — 2026-10-20

This inspection is read-only and covers the current service and PostgreSQL
repository paths. No implementation conclusion here means that concurrency is
already safe; the listed interleavings require FASE 21 integration coverage.

### Observed paths before FASE 21 implementation

| Operation | Current lock/read order |
|---|---|
| Deposit / withdrawal | Begin UOW → `SavingsAccounts.GetByIDForUpdate` (`FOR UPDATE`) → authorize account → check account OPEN → read active transactions → write transaction and audit → commit. No academic-year state lock/check is currently performed in these methods. |
| Year-end settlement | Begin UOW → savings account `FOR UPDATE` → authorize → check account OPEN → academic year plain `GetByID` (no row lock), require CLOSING → read ledger → update account, settlement, audit → commit. |
| Student-leaving settlement | Begin UOW → savings account `FOR UPDATE` → authorize → check account OPEN → read student and ledger → update account, settlement, audit → commit. No year row lock. |
| Settlement reopen | Read settlement without row lock → savings account `FOR UPDATE` → read academic year without row lock → read/update settlement and account, audit → commit. |
| Start year closing | Academic year `FOR UPDATE` → inspect pending transfer requests → update year → commit. |
| Close year | Academic year `FOR UPDATE` → check for unsettled accounts and pending transfer requests → update year → commit. The account existence check is a plain read, not a row lock. |
| Account creation trigger | On insert, academic year `FOR SHARE` → student `FOR SHARE` → validate eligibility. |
| Reopen-account trigger | On an account status transition back to OPEN, reads the academic year `FOR SHARE` and rejects if it is CLOSED. |

Primary sources inspected:

- `internal/service/transaction_service.go` (`Deposit`, `Withdrawal`)
- `internal/service/settlement_service.go` (`SettleYearEnd`,
  `SettleStudentLeaving`, `Reopen`)
- `internal/service/academic_year_service.go` (`StartClosing`, `Close`)
- `internal/repository/postgres/academic_year_repository.go`
  (`GetByIDForUpdate`)
- `internal/repository/postgres/savings_account_repository.go`
  (`GetByIDForUpdate`, `HasOpenByAcademicYear`)
- `migrations/000006_fase_16_8_contract_enforcement.up.sql`
  (account creation, reopen, and year-close constraints)
- `internal/repository/postgres/unit_of_work.go` (UOW transaction wiring)

### Lock-order analysis and required implementation behavior

The existing account-first transaction path serializes deposit/withdrawal
against settlement on the same account, but a plain year read would not
serialize a write against `StartClosing`. A transaction could otherwise
observe OPEN while the year transition commits to CLOSING. FASE 21 must add
coordination with the year row.

The normative order in §5 is compatible with the inspected lifecycle paths:

- `StartClosing`/`Close` take the year lock but do not then wait on a locked
  account row; a transaction holding the shared year lock can finish, after
  which closing proceeds. A transaction queued behind closing observes the
  resulting year state.
- Settlement and reopen take the account lock first. Settlement reads the year
  without a row lock; reopen also reads it without a row lock, then its account
  update trigger may take year `FOR SHARE`. Since transaction writes already
  hold `FOR SHARE` on the same year, those shared locks are compatible. If a
  transaction waits on the account, settlement/reopen can finish without
  waiting for the transaction's year lock, so this path does not create a lock
  cycle. After acquiring the account lock, the transaction must recheck status;
  settlement may have changed it to SETTLED. If year closing holds `FOR UPDATE`,
  the reopen trigger waits and then rejects a CLOSED year.
- The account's academic-year association must be rechecked after its lock is
  acquired, even though current lifecycle operations do not reassign it.
- The implementation adds `AcademicYearRepository.GetByIDForShare` and
  acquires that lock before the savings-account `FOR UPDATE` lock. It rechecks
  account status and academic-year association after locking the account.

The implementation and PostgreSQL integration coverage now demonstrate:

- A transaction holding the shared year lock completes before
  `StartClosing` can commit CLOSING.
- A transaction waiting behind a committed CLOSING transition is rejected.
- A concurrent settlement that wins the account lock causes the waiting
  transaction to observe SETTLED and fail; the inverse order lets the
  transaction commit before settlement reads the ledger.
- Concurrent deposits/withdrawals for one account remain serialized, and
  independent accounts remain independent.
- All above concurrency scenarios complete without deadlock against real
  PostgreSQL.

## 13. Implementation verification — 2026-10-20

The implementation and verification record is
[`fase-21-implementation-lock.md`](fase-21-implementation-lock.md). FASE 21
is locked at implementation commit `fe21e78`. No browser E2E run is claimed.

