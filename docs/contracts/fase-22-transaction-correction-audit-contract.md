# FASE 22 — Transaction Correction and Audit Contract

Status: LOCKED — product owner approved
Implementation status: IMPLEMENTED — verification passed; owner review and implementation lock record pending
Roadmap: Roadmap Resmi Tabungan Siswa v3, FASE 22
Dependencies: FASE 20 savings/passbook and FASE 21 transaction engine

This contract defines how a recorded transaction can be corrected without
erasing its history. Existing service methods are treated as a baseline to
review, not proof that the FASE 22 workflow is complete. The product-owner
decisions in §13 are confirmed. The correction workflow and entity audit
viewer are implemented and verified; owner review and the implementation lock
record remain pending.

## 1. Goal

Allow authorized Wali Kelas to correct an eligible transaction by editing its
permitted fields or cancelling it, while preserving a complete, readable audit
trail. Keep every account's chronological balance valid after each correction
and provide a scoped audit viewer for financial records.

## 2. Roadmap scope

Included:

- Edit an active transaction using only fields explicitly permitted in §4.
- Cancel an active transaction as a historical status change; never hard
  delete it.
- Recalculate the complete chronological active ledger after each proposed
  edit/cancel and reject any result with a negative chronological balance.
- Capture actor, action, timestamp, reason, entity, and before/after state for
  every correction.
- Keep transaction and audit changes atomic.
- Show active and cancelled entries clearly in transaction history; cancelled
  entries are excluded from balances but remain visible for traceability.
- Provide an authorized audit viewer for transaction corrections and relevant
  financial entity events.
- Preserve create events recorded by FASE 21 and settlement/reopen events
  covered by FASE 23.
- Add PostgreSQL integration, concurrency, authorization, and presentation
  coverage.

Not included:

- Hard deletion or rewriting creator identity/account ownership.
- Editing or cancelling transactions in an ordinary CLOSING/CLOSED year or a
  SETTLED account.
- Reopening settlements or years. These require the approved FASE 23 workflow.
- General-purpose audit administration, audit deletion, or audit retention
  changes.
- Student access to internal before/after snapshots or staff audit events.
- Changing deposit/withdrawal posting behavior already locked in FASE 21.

## 3. Inherited decisions

- The FASE 21 contract is authoritative for roles, Wali scope, date bounds,
  amount representation, canonical chronological order, complete-ledger
  validation, and non-negative balance rules.
- The effective-date interval is
  `academic_year.start_date <= transaction_date < academic_year.end_date`.
- Every operation derives the account and year through the transaction ID;
  client-supplied ownership IDs never replace this relationship.
- ADMIN is read-only for financial mutation under
  [`financial-supervision-access.md`](financial-supervision-access.md).
- WALI_KELAS retains financial operations only within the historically valid
  assigned class/year scope. SISWA has read-only access to their own account.
- Audit entries are append-only evidence. A later correction creates another
  event and never overwrites an earlier event.
- FASE 23's locked contract defines a future approved reopen flow for
  correction of a settled account or a non-OPEN academic year. FASE 22 does
  not implement that flow or expose a general bypass; FASE 23 implementation
  must call the correction rules defined here and apply its own approved-scope
  authorization.

## 4. Correction semantics

### Edit

- Only an ACTIVE transaction may be edited.
- Only amount and effective transaction date are editable. Type, account,
  student, academic year, creator, status, and ID are immutable through the
  edit form.
- Amount remains a positive whole-rupiah value within domain/database limits.
- Effective date remains within the owning academic year's start-inclusive,
  end-exclusive period.
- Recalculate the full ACTIVE ledger using the candidate values. The edit is
  rejected if any chronological balance becomes negative, including balances
  after the edited transaction's date.
- Preserve the original `created_by` and `created_at`. Update the current
  transaction row and `updated_at`; the audit event records the correcting
  actor and prior/new values.

### Cancel

- Cancel is a soft historical transition from ACTIVE to CANCELLED. No DELETE
  statement or hard-delete route is allowed.
- Cancellation is terminal for that transaction. Restoring a cancelled item
  uses a new, separately audited transaction; it does not flip the old row back
  to ACTIVE.
- Recalculate the full remaining ACTIVE ledger. Reject cancellation if removing
  that transaction makes any later chronological balance negative.
- Retain cancelled transaction data and its audit trail. Exclude it from the
  passbook balance but display it as cancelled in transaction history.

### Reason

- Every edit and cancellation requires a non-empty reason after trimming
  whitespace, bounded to 500 Unicode characters. Render the reason in the
  audit timeline and preserve it in the audit snapshot.
- The reason explains the correction; it does not authorize a correction that
  violates scope, lifecycle, or ledger rules.

## 5. Lifecycle and scope

For ordinary FASE 22 corrections, all of these conditions must hold at the
serialized decision point:

1. Actor is an authenticated WALI_KELAS.
2. The transaction exists, belongs to an OPEN savings account, and belongs to
   an OPEN academic year.
3. The actor is assigned to the transaction's student/class/year scope.
4. The transaction is ACTIVE.
5. The resulting chronological ledger remains valid and non-negative.
6. The correction reason and permitted fields pass validation.

ADMIN cannot edit/cancel even though Admin can read the transaction and its
audit. SISWA cannot edit/cancel. Unknown resources preserve established
not-found behavior; do not leak cross-class existence.

FASE 22's normal edit/cancel operations require an OPEN account in an OPEN
academic year. During a FASE 23 approved correction window, the responsible
Wali may edit/cancel only the approved account's transactions while the year
is CLOSING and that account is OPEN. The service accepts this exception only
through an internal context created by the FASE 23 settlement workflow after
validating the approved request; no HTTP field or route can create that scope.
Ordinary edits, cancellations, deposits, and withdrawals remain blocked
during the correction window. FASE 22 does not authorize settlement reopen
itself.

## 6. Audit contract

Each event records:

- `who`: authenticated actor user ID and safe display name when rendered.
- `what`: stable action name (`CREATE` from FASE 21, `UPDATE`, `CANCEL`, and
  future settlement/reopen actions from FASE 23).
- `when`: server-generated timestamp, not client-supplied time.
- `which entity`: entity type and immutable entity ID, with a link to the
  owning transaction/account/year where authorized.
- `before`: full permitted prior transaction state for UPDATE/CANCEL.
- `after`: full resulting transaction state plus the correction reason.
- The CREATE event remains attributed to its original creator. UPDATE/CANCEL
  events identify the correcting actor; they never rewrite the creator.

Audit writes for UPDATE/CANCEL are in the same PostgreSQL transaction as the
transaction update. If the audit insert or transaction commit fails, the
correction rolls back and the UI reports no success. Audit rows are append-only
for the application workflow.
For a correction performed during an approved FASE 23 window, the after
snapshot also records the approved correction request ID.

Audit data and logs must never contain passwords, password hashes, cookies,
session secrets, CSRF tokens, credentials, or raw request bodies. Store only
allow-listed transaction fields and the bounded reason. The viewer renders
human-readable labels, not unsanitized JSON or SQL/database internals.

## 7. Audit viewer authorization

| Capability | ADMIN | WALI_KELAS | SISWA |
|---|---|---|---|
| View transaction/account audit | Any financial entity | Entity belonging to assigned class/year scope | Deny; regular account/passbook reads remain available |
| Search audit by actor across system | Yes, if exposed in a later approved administration scope | Deny | Deny |
| Create/edit/cancel transaction | Deny | Assigned class/year only, subject to §5 | Deny |

For WALI_KELAS, authorization is checked through audit entity → transaction or
account → student → historical class placement → class assignment. An audit
actor ID is never an authorization grant. No role may access a log by guessing
an audit ID. This phase's UI is entity-scoped; global actor search is not
included in the FASE 22 routes.

## 8. Concurrency and persistence

- Each edit/cancel executes in one PostgreSQL transaction.
- Use the FASE 21 lock order: read the target transaction/account only to
  discover the immutable academic-year ID; lock that year `FOR SHARE`; verify
  lifecycle/date; lock the savings account `FOR UPDATE`; authorize scope; then
  re-read the target transaction and the full active ledger under the same
  transaction.
- Hold the year and account locks through validation, transaction update,
  audit insert, and commit. This serializes corrections with deposits,
  withdrawals, other corrections, settlement, and year close without
  introducing an account-then-year lock cycle.
- Concurrent edit/edit, edit/cancel, cancel/cancel, and correction/deposit or
  correction/withdrawal on one account serialize. A request queued behind
  another mutation re-reads the committed transaction and account state before
  deciding.
- Independent accounts do not share an account lock and should proceed
  independently, except for their common shared academic-year lock.
- Affected-row checks must detect a stale/missing row. A repeated cancel
  returns a safe conflict and causes no extra audit event.
- No migration is presumed. Inspect audit constraints and transaction status
  constraints before adding schema. Any migration needs tested upgrade and
  rollback evidence.

## 9. HTTP and presentation surface

All routes require authentication. GET supports HEAD. POST requires CSRF,
bounded URL-encoded form validation, and actor identity from authenticated
context. Service authorization and database checks remain authoritative.

| Method | Path | Purpose |
|---|---|---|
| GET | `/transactions/{transaction_id}` | Current transaction details and authorized audit timeline |
| POST | `/transactions/{transaction_id}/edit` | Edit an active transaction's permitted fields |
| POST | `/transactions/{transaction_id}/cancel` | Soft-cancel an active transaction |

The edit body is `amount`, `transaction_date`, `reason`, `csrf_token`.
The cancellation body is `reason`, `csrf_token`. The request must not accept
actor, account/year/student IDs, transaction type, creator, status, balance,
audit event, or timestamp from the browser.

Transaction history displays current ACTIVE values and retained CANCELLED
entries with a clear status badge. For each authorized transaction, the detail
view shows a chronological audit timeline with actor, action, timestamp,
reason, and before/after field changes. Students continue to see their
financial transaction history but do not see staff-only correction reasons or
audit snapshots.

## 10. Validation, redirects, and error mapping

Validation order for POST:

1. Authentication and method.
2. Content-Type, request-size, and form-encoding validation.
3. CSRF validation.
4. Route ID and submitted field syntax/range validation.
5. Service lookup, role/scope, account/year lifecycle, transaction status,
   reason, date range, and complete-ledger validation.
6. Atomic update and audit commit; redirect only after success.

| Condition | Response |
|---|---|
| Detail/timeline success | 200 HTML |
| Edit/cancel success | 303 to transaction detail/history with safe notice |
| Missing/invalid authentication | 401 |
| Unsupported method | 405 with `Allow` |
| Malformed/oversized body, invalid ID/date/amount/reason | 400 |
| Missing/invalid CSRF | 403 |
| Forbidden role or class/year scope | 403 |
| Unknown transaction | 404 |
| Cancelled/stale transaction, closed lifecycle, negative chronological balance, or already-cancelled item | 409 with actionable safe explanation |
| Unexpected failure | 500 generic public message; safe structured logging only |

## 11. Verification and exit criteria

FASE 22 may be locked only when all applicable evidence passes:

- Unit/service coverage for permitted fields, reason validation, roles/scope,
  active/cancelled transitions, date range, complete-ledger recalculation,
  non-negative balance, repeat requests, and audit atomicity.
- PostgreSQL integration tests for edit/cancel behavior, audit rollback,
  balance preservation, row locking, concurrent correction/deposit/withdrawal,
  and independent accounts.
- HTTP/CSRF/template tests for auth, bounded forms, malformed input, safe
  redirects/notices, scoped audit visibility, Admin read-only behavior,
  student audit denial, active/cancelled history, and before/after rendering.
- Tests prove audit before/after values and actor/reason/timestamp match each
  committed correction and that failed mutations leave no audit entry.
- `make test-integration`, race-enabled PostgreSQL integration tests,
  `go vet ./...`, `go build ./...`, frontend tests, and `git diff --check` pass.
- Migration, if any, has tested upgrade/rollback evidence.
- Browser E2E is claimed only if actually run.
- Implementation commit and FASE 22 lock record are created after owner
  review.

## 12. Current implementation inspection

Baseline inspection before FASE 22 implementation on 2026-10-20 found:

- `TransactionService.Edit` and `Cancel` already exist in the service layer.
  They are available only to WALI_KELAS and write UPDATE/CANCEL audit events.
- Edit currently accepts amount and transaction date; Cancel is a soft status
  transition. Both recalculate the active ledger and use the account lock.
- Edit acquires the year `FOR SHARE` before the account lock; Cancel currently
  does not lock/check the academic-year row. The locked FASE 21 order must be
  applied consistently to both operations.
- The Edit service currently validates the date range but does not require the
  academic year to be OPEN. FASE 22 must enforce lifecycle state for ordinary
  corrections and reserve a narrow, approved FASE 23 correction path.
- Audit service/repository list APIs exist; `ListByEntity` and `ListByActor`
  currently authorize ADMIN only. There is no transaction edit/cancel HTTP
  route or transaction audit UI.
- Existing service tests cover some edit/cancel ledger behavior, but do not
  satisfy the complete HTTP, scoped viewer, audit snapshot, and concurrency
  criteria in §11.

## 13. Product-owner decisions — confirmed 2026-10-20

The product owner approved these rules for implementation:

1. Every edit and cancellation requires a non-empty reason after trimming
   whitespace, with a maximum length of 500 Unicode characters. The reason is
   preserved in the immutable audit event and shown in the authorized audit
   timeline.
2. An edit may change only the transaction amount and effective date. Type,
   account, student, academic year, creator, status, and identifier are
   immutable. Correct a wrong transaction type by cancelling the old entry
   and recording a separately audited transaction.
3. Admin may read financial audit records across all scopes. Wali Kelas may
   read transaction and account audit only within the assigned class/year
   scope. Students retain ordinary passbook access but cannot read internal
   audit snapshots or staff correction reasons. Actor-wide audit search is
   Admin-only and outside this phase's UI.
4. Ordinary corrections require both the savings account and academic year to
   be OPEN. The sole exception is the narrowly approved correction window
   defined by FASE 23; regular deposits and withdrawals remain blocked during
   that window.

## 14. Pre-implementation review checklist

- [x] Roadmap v3 FASE 22 scope and FASE 21 boundary reviewed.
- [x] Current transaction, audit, role, and access-policy implementation
      inspected.
- [x] Backdated-date and non-negative-ledger semantics inherited from the
      locked FASE 21 contract.
- [x] Product-owner decisions in §13 confirmed.
- [x] Contract status set to LOCKED before FASE 22 code changes begin.
