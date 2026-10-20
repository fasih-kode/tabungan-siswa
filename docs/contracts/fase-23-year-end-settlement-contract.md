# FASE 23 — Year-End Settlement Contract

Status: LOCKED — product owner approved  
Implementation status: IN PROGRESS — local working-tree implementation; review and verification pending
Roadmap: Roadmap Resmi Tabungan Siswa v3, FASE 23  
Dependencies: FASE 18 academic-year lifecycle, FASE 20 savings cycles, FASE 21 transactions, FASE 22 correction and audit

This contract defines the year-end settlement, monitoring, close, and
controlled reopen workflows. Contract decisions are locked. Implementation
work is in progress and is not considered accepted until review and required
verification are complete.

## 1. Goal

Let assigned Wali Kelas review and settle each eligible savings account while
an academic year is closing; let Admin monitor completion, see exactly which
Wali Kelas/classes/students still block closure, and close the year only after
all required work is complete. Preserve an auditable financial history and do
not carry a prior year's balance into a new year's account.

## 2. Scope

Included:

- Closing-balance preview for an account, calculated from its complete active
  transaction ledger.
- Per-account year-end settlement and account transition from OPEN to SETTLED.
- Settlement detail and history, including amount, type, actor, and time.
- Separate confirmation that the settlement amount was handed to the student,
  including who confirmed it and when.
- A year-level readiness view in the existing **Pengawasan Tabungan** area.
- Admin monitoring grouped by the responsible Wali Kelas, with outstanding
  account count and drill-down to class, student, account, balance, and
  passbook/settlement details.
- Explicit unresolved-assignment groups for classes with no responsible Wali
  Kelas or student accounts that cannot be attributed to a class.
- Academic-year close after all settlement and class-transfer blockers are
  cleared.
- Reopen request, approval, controlled correction, re-settlement, and
  re-closing workflow as defined in §14.
- Atomic persistence and audit records for every accepted state change.
- PostgreSQL integration and concurrency coverage.

Not included:

- Editing/cancelling transactions except through the locked FASE 22 correction
  contract and an approved FASE 23 reopen workflow.
- Automatic transfer/carry-over of balances to another academic year.
- Bank, payment-gateway, or cash-register integration.
- General financial reports/dashboard owned by FASE 25.
- Student-leaving settlement policy changes. Existing student-leaving
  settlements remain available under their own lifecycle rules; they count as
  already settled for year-close readiness.

## 3. Terms and invariants

- **Year-end settlement**: an immutable completed settlement record for one
  savings account and academic year, using the account's closing ledger
  balance.
- **Outstanding account**: an account in the selected academic year whose
  status is OPEN and which therefore still prevents year close.
- **Responsible Wali Kelas**: the teacher assigned to the student's class for
  the selected academic year at its final in-year date. Since year date ranges
  are start-inclusive/end-exclusive, this attribution uses the last calendar
  date inside the year. Do not use the student's current class or a current
  teacher assignment to rewrite historical responsibility.
- **Unattributed account**: an outstanding account with no valid in-year class
  placement, or whose historical class has no Wali Kelas assignment on the
  attribution date. Show it in an explicit “Belum ada penanggung jawab” group;
  never hide it or guess an owner.
- **Year close readiness**: ready only when the year is CLOSING, there are no
  outstanding accounts, every positive settlement has a recorded handover
  confirmation, there are no pending class-transfer requests, and no approved
  reopen/correction/re-settlement workflow is active.
- Account and settlement history is retained. Reopening supersedes a previous
  settlement; it never deletes or silently overwrites it.
- Students without a savings-account cycle for the selected year have no
  settlement obligation and do not block year close.
- A new academic-year account starts at zero. No amount, transaction, or
  settlement is copied automatically from the prior year.
- Once a successor academic year exists, the predecessor is not reopenable
  through the normal UI or service workflow. A reopen request for it must be
  rejected, and Admin cannot approve an existing request after the successor
  is created.

## 4. Ledger and preview semantics

- Balance is recomputed from every ACTIVE transaction, never from a page of
  results, cached balance, or previously stored preview.
- Use the FASE 21 canonical order: effective transaction date, transaction
  creation timestamp, then transaction UUID; apply deposits and withdrawals
  chronologically and reject invalid/negative ledger states.
- The academic-year effective-date interval remains the locked FASE 21
  start-inclusive/end-exclusive interval.
- Preview is read-only and is recalculated on every request. It displays the
  calculated amount and the transaction cutoff/preview time so the user knows
  what is being confirmed.
- A settlement captures the balance calculated inside the write transaction
  after the required locks are acquired. The stored amount must equal the
  preview if the ledger has not changed; if it has changed, render the updated
  amount and require explicit confirmation before creating the settlement.
- A zero balance is a valid settlement amount and still transitions the
  account to SETTLED. Since no money is due, mark handover as
  `NOT_REQUIRED_ZERO_AMOUNT`; do not ask the Wali to confirm a payment that
  did not occur.
- For a positive settlement, account settlement and money handover are
  separate states. Settling the account records the closing obligation;
  year-close readiness remains blocked until the assigned Wali confirms the
  student received the full settlement amount.
- Handover confirmation identifies the account's student as the recipient,
  records the exact settlement amount, confirming Wali, and confirmation
  timestamp. The server supplies the recipient, amount, and timestamp; the
  confirmation cannot change them. A receipt/reference is optional and is not
  required by this approved decision.
- This confirmation requirement applies to every positive completed
  settlement for the selected year, including a student-leaving settlement.
  Existing positive settlements without handover evidence must be visibly
  marked “Konfirmasi penyerahan belum tercatat” and reconciled by an authorized
  Wali before the year can close; do not infer that cash was handed over.
- A completed student-leaving settlement already attached to the account
  satisfies the settlement requirement. Do not create a second completed
  settlement for that account.

## 5. Roles and authorization

Authorization follows the current [financial supervision policy](financial-supervision-access.md).

| Capability | ADMIN | WALI_KELAS | SISWA |
|---|---|---|---|
| View year readiness and all outstanding-account groups | Yes, all years/classes | Own assigned classes only | No |
| Read account preview, settlement, and passbook | Any account/year | Accounts in assigned class/year scope | Own linked student's account only |
| Execute regular year-end settlement | Deny | Responsible Wali for the historical class/year only; explicit preview confirmation is the Wali approval | Deny |
| Confirm student handover for a positive settlement | Deny | Responsible Wali for the historical class/year only | Deny |
| Request settlement reopen | Deny | Responsible Wali for the settlement's historical class/year only | Deny |
| Approve/reject reopen request | Admin only; requester cannot approve | Deny | Deny |
| Execute approved correction and re-settlement | Deny | Responsible Wali, through FASE 22 correction and approved reopen scope | Deny |
| Close academic year | Admin only, using existing lifecycle route/service | Deny | Deny |

Every service operation derives authorization from the authenticated actor and
the account → student → in-year placement → class → teacher assignment chain.
Submitted IDs, query filters, creator IDs, and display grouping never grant
access. Admin's read-only monitoring permission does not grant Admin financial
write permission.

## 6. Admin monitoring requirement

The existing **Pengawasan Tabungan** menu is the entry point; this phase adds a
year-level settlement-readiness view there rather than making Admin infer
readiness from one student's passbook at a time.

For a selected year, Admin sees:

- Year name and status, readiness state, and counts of total, outstanding,
  settled, and unattributed accounts.
- A separate count and explanation for pending class-transfer requests, since
  they are also a close blocker.
- One summary row per responsible Wali Kelas and class, containing the Wali
  display name, class name, outstanding account count, and link to the scoped
  outstanding-account detail list.
- Per-Wali counts distinguish accounts still OPEN, settled but awaiting
  student handover, and fully completed. A positive settlement awaiting
  handover remains a year-close blocker.
- An explicit unassigned group for every outstanding account without a
  responsible Wali Kelas.
- Outstanding-account detail rows with student name, student number where
  available, class, account status, current ledger balance, and a link to the
  read-only book/transaction detail.
- A settled list/history so Admin can verify completed work and its actor/time.
- An empty/ready state when no settlement or transfer blockers remain.

Counts are distinct savings accounts, not transaction counts. An account is
outstanding only while its account status is OPEN. Do not count a settled
account again because an old settlement was superseded unless the account has
been reopened and is currently OPEN. Do not omit accounts whose student is
inactive, graduated, or left; show their status and outstanding state.

The attribution date and historical class/assignment source are defined in §3.
If placement/assignment data are missing or ambiguous, show an unresolved data
issue; do not attribute the account to the current teacher. Ambiguous matches
must remain a single account row, count as unattributed, and block readiness.
Admin can still open the student/account detail under existing read scope.
Wali readiness counts pending transfers attributed to classes for which that
Wali is responsible at the same historical year-end date; it must not expose
the whole-year count to an individual Wali.

## 7. Settlement workflow

1. Admin starts year closing through the existing academic-year lifecycle.
   The existing rule rejecting pending class transfers remains authoritative.
2. The year enters CLOSING. Regular deposits/withdrawals are unavailable under
   the FASE 21 OPEN-year rule.
3. The Wali Kelas responsible for the class on the final in-year date opens an
   outstanding account, reviews the preview and passbook, then explicitly
   confirms settlement.
4. The service recalculates the balance under locks, creates a
   `REGULAR_YEAR_END` settlement, changes the account from OPEN to SETTLED, and
   writes settlement and account audit events atomically.
5. For a positive settlement, the responsible Wali records handover only after
   the student receives the full amount. The server binds the confirmation to
   the account's student and settlement amount; the Wali cannot submit a
   different recipient or amount. A zero settlement is marked
   `NOT_REQUIRED_ZERO_AMOUNT`.
6. The Wali sees a receipt showing settlement and handover status separately,
   amount, recipient student, confirming actor, timestamps, and settlement
   detail. Retrying either completed action cannot duplicate or change it.
7. Admin monitors progress in Pengawasan Tabungan. The readiness summary
   updates from persisted state; it must not rely on client-submitted counts.
8. Admin closes the academic year only when the service's authoritative checks
   confirm readiness. A concurrent settlement or transfer request cannot
   cause a false ready/closed state.

If settlement persistence, account update, or any required audit insertion
fails, the complete operation rolls back and no success is reported.

## 8. Close readiness and lifecycle

- Existing lifecycle states and routes remain authoritative: OPEN → CLOSING →
  CLOSED. This phase does not add a shortcut from OPEN to CLOSED.
- Admin readiness information may be viewed before and during closing. A year
  is not ready while OPEN; Admin must first start closing.
- The year service remains the final authority. It must atomically lock the
  year, recheck that it is CLOSING, verify no OPEN savings accounts and no
  positive settlement awaiting handover, verify no pending class transfers,
  transition it to CLOSED, and commit.
- The monitoring page is advisory; client-side state or cached totals cannot
  authorize close.
- Existing rule remains: a subsequent academic year can be created only after
  all existing years are CLOSED. Once created, the new year's balances remain
  independent from the old year.
- Do not create a successor while a reopen request, approved correction,
  re-settlement, or handover reconciliation for the predecessor is active.
  The year-create service must check this authoritative persisted state as
  well as academic-year status.
- Reopen request/approval and successor creation must serialize on the
  predecessor academic-year row. If successor creation commits first, reject
  the request/approval; if an approved correction window commits first, block
  successor creation until the correction is re-settled and the year is
  closed again.
- Once a successor exists, neither a new reopen request nor an old pending
  request may reopen the predecessor. Reopen approval must recheck this
  condition at execution time, not rely on the earlier request check.
- If close is rejected, return the user to the year detail with a safe reason
  and a link to the relevant outstanding account or transfer list.

## 9. Reopen, correction, and re-settlement

The product owner confirmed this workflow: the responsible Wali Kelas requests
reopening with a reason; Admin approves or rejects the request; after approval,
the responsible Wali Kelas carries out the correction through FASE 22 and then
re-settles the account. Admin approves but does not directly mutate the
settlement, account, or transaction. Approval starts a system-controlled
correction window that moves the year back to CLOSING and reopens only the
approved account; it does not grant Admin authority to change money or
transactions. FASE 23 must not expose a bypass that edits the ledger without
FASE 22 authorization and audit.

The reopen flow must:

- Capture requester, target settlement/account/year, reason, request time,
  decision actor, decision time, and decision rationale.
- Permit only the Wali Kelas responsible for the historical class/year to
  request reopening. Permit only one active pending request per settlement;
  concurrent duplicate requests do not create duplicate approvals.
- Require a distinct authorized Admin decision; requesters cannot approve
  their own request.
- Preserve the old completed settlement as SUPERSEDED and link the replacement
  settlement through `replaces_settlement_id`.
- Limit the approved correction window to the approved account/year and record
  every affected transaction and settlement action in the audit trail.
- Require a new explicit settlement confirmation after correction. The new
  settlement amount is calculated from the corrected complete ledger, not
  copied from the previous settlement. Reconcile any difference from money
  already handed to the student according to confirmed decision 6 in §14.
- Re-confirm handover for any additional amount given to the student and
  record any returned amount as an audited recovery. Preserve the original
  handover confirmation; never rewrite it to make the new settlement appear
  to have been distributed earlier.
- Re-run the year close readiness checks before closing again.

This approved workflow amends the prior policy, which intentionally authorized
no role to reopen settlements. It grants no direct financial mutation to
Admin. The successor-year restriction and the financial treatment of a
re-settlement difference are recorded in §14.

## 10. HTTP and presentation surface (locked)

All routes require authentication. GET supports HEAD. Every POST requires
CSRF, a bounded URL-encoded form, and actor identity from authenticated
context. Services enforce authorization and lifecycle rules.

| Method | Path | Purpose |
|---|---|---|
| GET | `/savings?academic_year_id={id}&view=settlement` | Existing Admin/Wali supervision area with readiness and outstanding groups |
| GET | `/savings/{account_id}/settlement-preview` | Read-only current closing-balance preview |
| POST | `/savings/{account_id}/settlements` | Confirm regular year-end settlement |
| GET | `/settlements/{settlement_id}` | Read settlement detail and history context |
| POST | `/settlements/{settlement_id}/reopen-requests` | Submit reopen request |
| POST | `/settlement-reopen-requests/{request_id}/decision` | Admin approve/reject with reason |
| POST | `/settlements/{settlement_id}/reopen` | Execute approved reopen and correction window |
| POST | `/settlements/{settlement_id}/distribution-confirmation` | Confirm handover of the settlement amount to the account's student |
| POST | `/academic-years/{id}/close` | Existing route; service performs authoritative readiness check |

The route shape and role semantics above are locked. Any change requires an
explicit contract amendment before implementation.

Settlement confirmation accepts only `csrf_token` and a preview version/token
or equivalent server-verifiable preview fingerprint. It must not accept an
amount, account status, year status, actor, or balance from the browser as
authoritative. If a preview token is not used, the confirmation screen must
re-display the locked amount and require confirmation against that server-side
state.

Distribution confirmation accepts `csrf_token` and an explicit
`confirm_handover=1` acknowledgement only. The server derives the settlement,
amount, student recipient, historical responsible Wali, and timestamp from
persisted state and the authenticated actor.

## 11. Response and error behavior

| Condition | Response |
|---|---|
| Readiness, preview, or detail success | 200 HTML |
| Settlement/reopen request/decision success | 303 redirect to relevant detail |
| Missing/invalid authentication | 401 |
| Invalid UUID/form/date/page/CSRF | 400 or 403 per shared middleware contract |
| Role/scope denial | 403 |
| Resource not found | 404 |
| Wrong year/account/settlement lifecycle or already completed | 409 with safe, actionable explanation |
| Stale preview, conflicting approval, or concurrent state change | 409; reload authoritative state |
| Unsupported method | 405 |
| Unexpected failure | 500 with generic public message; safe structured logs only |

Do not reveal whether an out-of-scope account or settlement exists. Do not
render internal SQL, stack traces, credentials, cookies, session secrets, or
CSRF values in errors/logs.

## 12. Persistence, audit, and concurrency

- Every settlement, distribution confirmation, reopen/approval, recovery, and
  close state change is one database
  transaction with required audit writes; any failure rolls back all changes.
- Settlement and handover audits record actor, action, entity, timestamp, type, amount,
  account/year identifiers, and before/after state. Reopen-request audit also
  records requester reason and Admin decision/rationale; distribution audit
  records the account's student recipient. Never accept the recipient or amount
  as authoritative browser input and never record secrets.
- Existing audit history is append-only for this workflow. A correction never
  silently overwrites prior settlement evidence.
- Transaction write lock order from FASE 21 is normative: discover immutable
  account year, lock the academic-year row `FOR SHARE`, then lock the savings
  account `FOR UPDATE`; retain locks through ledger read, mutation, audit, and
  commit. Settlement writes must use a compatible year-before-account order.
- Academic-year close keeps its year `FOR UPDATE` lock before the final
  outstanding-account and transfer checks. This serializes final close with
  account settlement, which must hold the compatible year `FOR SHARE` lock.
- Reopen request/approval and successor creation must take the predecessor
  academic-year lock before reading/writing reopen-request state. This prevents
  an approval from racing past a committed successor-year insert and prevents
  successor creation from passing an active approved correction.
- Reopen must use the same year-before-account order and must never introduce
  an account-then-year lock cycle with FASE 21, FASE 22, account triggers, or
  year close. Final lock order is subject to PostgreSQL integration proofs.
- Settlement balance is read from the full active ledger while the account
  lock prevents concurrent mutation. Do not depend on a prior preview as the
  source of truth.
- Database uniqueness/constraints remain the final defense against multiple
  completed settlements for one account and invalid account/year state.
- A schema migration is not presumed. Inspect current constraints, audit
  schema, and request/approval persistence needs; document upgrade/rollback if
  a migration is necessary.

## 13. Verification and exit criteria

FASE 23 may be locked only when all applicable evidence passes:

- Unit/service coverage for role matrix, class/year scope, lifecycle, zero and
  positive balance, duplicate settlement, stale preview, audit rollback,
  transfer blockers, and reopen-request decisions.
- PostgreSQL integration coverage for ledger snapshot consistency,
  settlement/withdrawal and settlement/year-close interleavings, duplicate
  concurrent settlement, reopen/correction/re-settlement history, and absence
  of deadlocks.
- HTTP/CSRF/template tests for Admin overview, Wali scoped workflows, student
  read-only history, unassigned Wali groups, missing placement, empty/ready
  states, blockers, and safe errors.
- Tests prove Admin cannot settle, reopen, correct, or approve their own
  request by direct route/service invocation beyond the approved decision
  policy.
- Tests prove outstanding account counts and class/Wali attribution are
  derived server-side from the selected historical academic year.
- `make test-integration`, the established race-enabled PostgreSQL integration
  command, `go vet ./...`, `go build ./...`, frontend tests, and
  `git diff --check` pass.
- Migration, if any, has tested upgrade and rollback evidence.
- Browser E2E is reported only if actually run.
- Implementation commit and FASE 23 lock record are created after owner review.

## 14. Product-owner decisions

### Additional decisions confirmed — 2026-10-20

1. **Student handover**: the system records confirmation that money was handed
   to the student. FASE 23 therefore treats settlement and handover as separate
   states, records the confirming Wali and timestamp, and blocks year close
   while a positive settlement lacks handover confirmation. A zero amount
   requires no handover.
2. **Reopen and correction**: the responsible Wali Kelas requests reopening;
   Admin approves or rejects; after approval, the Wali Kelas performs the
   correction through FASE 22, then re-settles. Admin does not execute
   financial mutations.
3. **Historical responsibility**: determine the responsible Wali from the
   assignment active on the final in-year date, paired with the student's
   historical class placement on that date. Missing/ambiguous data appears in
   an unresolved group and blocks close until resolved.
4. **Successor-year restriction**: a predecessor academic year cannot be
   reopened after a successor academic year has been created. Reject both new
   reopen requests and approvals that race with successor-year creation.

### Confirmed — 2026-10-20

5. **Academic-year state during approved correction**: before a successor
   exists, Admin approval moves the year from CLOSED back to CLOSING and
   reopens only the approved account. FASE 21 regular deposits and withdrawals
   remain unavailable because the year is not OPEN. FASE 22 accepts only the
   approved correction for that account. Successor-year creation is blocked
   while correction/re-settlement is active. After correction, the Wali
   re-settles and reconciles handover; Admin closes the year again.
6. **Cash delta after re-settlement**: preserve original handover evidence and
   record only the net additional handover or return needed to make cumulative
   cash delivered equal the corrected settlement amount. Each movement is
   confirmed by the responsible Wali and audited. Year close remains blocked
   while the net amount is unreconciled.
7. **Normal year-end approval**: the responsible Wali's explicit preview
   confirmation is the approval for that account's normal settlement. No
   separate per-account Admin approval is required; Admin monitors and performs
   final year close, consistent with the existing read-only financial policy.

## 15. Pre-implementation review checklist

- [x] Admin monitoring requirement added: pending accounts grouped by Wali and
      class, with counts and drill-down from Pengawasan Tabungan.
- [x] Roadmap FASE 23 scope and FASE 22 dependency reviewed.
- [x] Current financial supervision policy reviewed.
- [x] Existing settlement, year-close, and transaction-lock implementation
      inspected; implementation gaps are stated as requirements, not assumed
      complete.
- [x] Product-owner decisions recorded: settlement handover confirmation,
      reopen request/approval/correction roles, historical Wali attribution,
      and no predecessor reopen after successor-year creation.
- [x] All product-owner decisions in §14 confirmed or resolved consistently
      with the existing locked financial access policy.
- [x] Contract set to LOCKED before FASE 23 code changes begin.
