# Amendment — Student-Leaving Settlement Web Workflow

Status: APPROVED by product owner, 2026-10-21.

> **Urutan dalam amendment ini telah superseded.** Keputusan MQ-08-CONTRACT-02 mewajibkan settlement dan penyerahan yang diperlukan selesai sebelum siswa menjadi `LEFT`. Gunakan [amendment penyelesaian keuangan sebelum pengesahan pindah](student-transfer-financial-clearance-amendment.md) sebagai kontrak terbaru untuk memulai proses, settlement, dan finalisasi. Isi di bawah dipertahankan sebagai catatan keputusan/alur UI terdahulu, bukan perilaku yang berlaku untuk urutan baru.

This amendment adds the missing browser workflow for the existing
`STUDENT_LEAVING` settlement operation. It clarifies the separation already
specified by the [FASE 19 student contract](fase-19-student-http-contract.md)
and the handover requirement in the [FASE 23 settlement contract](fase-23-year-end-settlement-contract.md).

## Approved workflow

1. Admin marks the student as `LEFT` using the existing Pindah Sekolah action.
   This closes the student's class history but does not settle the savings
   account.
2. Wali opens **Tabungan** and sees any `OPEN` account for a `LEFT` student in
   the selected year as **Perlu penyelesaian**. The account row and account
   detail link to a dedicated leaving-settlement preview.
3. Wali reviews the full active transaction ledger and the previewed balance,
   then explicitly confirms the settlement. The write operation recalculates
   the balance under the existing account/year locks and rejects a stale
   preview; the user must review and confirm the latest balance again.
4. A positive settlement requires the Wali to confirm actual handover on the
   settlement detail. A zero-balance settlement does not require handover.
5. Admin monitors the resulting status from year-end readiness. The leaving
   flow remains separate from the Admin's **Mulai Penutupan** action.

## Roles and lifecycle constraints

- Admin remains responsible for changing student status and monitoring; Admin
  cannot execute financial settlement or handover.
- Only a Wali authorized for the student and academic-year history can preview,
  settle, and confirm handover.
- Leaving settlement is available while the academic year is `OPEN` or
  `CLOSING`, so a student departure can be resolved before year-end close.
  It is unavailable after the year is `CLOSED`.
- Settlement and handover retain the existing ledger, audit, concurrency, and
  account-scope checks. No migration is required.

## Approved authorization and status alignment — 2026-10-21

- Only a Wali Kelas may create deposits or withdrawals. Admin remains
  read-only for financial records as specified in
  [financial supervision access](financial-supervision-access.md).
- A new deposit or withdrawal requires the student to be `ACTIVE` at the
  serialized write point. For this check, the transaction locks and reads the
  student row so an Admin `LEFT` transition and a concurrent transaction
  cannot both succeed based on stale status.
- A `LEFT` student cannot receive new deposits or withdrawals. While the
  account and year permit corrections, the responsible Wali may still edit or
  cancel existing transactions under FASE 22's reason, scope, balance, and
  audit rules before settlement.
- The student-leaving preview and settlement apply only to `LEFT`. A
  `GRADUATED` student's open account is resolved by the regular academic-year
  closing workflow; the savings page explains this and does not label it as a
  school transfer.
- A Wali sees the transfer-settlement action only for a `LEFT` student's open
  account when the year is `OPEN` or `CLOSING`. If the year is `CLOSED`, the
  page shows that settlement is unavailable instead of linking to a failing
  action. Backend checks remain authoritative.
- A transaction missing from the ledger after departure has no exception
  entry workflow in this amendment. It must not be entered by reactivating the
  student; a separate audited procedure needs a product decision.
