# FASE 16.1 — DOMAIN CONTRACT AUDIT

Status: AUDIT COMPLETE
Phase: 16.1
Parent: FASE 16 — CONTRACT HARDENING
Scope: Domain model and domain invariants
Source baseline: commit `1a680c8`
Audit basis: source snapshot `tabungan-siswa.tar(20261008-034155).gz`

---

# 1. Tujuan

FASE 16.1 melakukan audit terhadap kontrak domain aplikasi Tabungan Siswa.

Audit membandingkan:

1. Domain entity.
2. Domain value object.
3. Domain state transition.
4. Domain validation.
5. Service-level enforcement yang secara langsung mewujudkan kontrak domain.
6. Database constraint yang menjadi enforcement terakhir.
7. Business contract yang telah ditetapkan sebelumnya.

Audit ini TIDAK melakukan perubahan kode.

Setiap temuan diklasifikasikan menjadi:

- `MATCH`
- `GAP - IMPLEMENTATION`
- `GAP - CONTRACT DECISION`
- `CONFLICT`
- `DEFERRED`

---

# 2. Prinsip Audit

Kontrak domain dibagi menjadi tiga lapisan:

```text
BUSINESS CONTRACT
        ↓
DOMAIN CONTRACT
        ↓
IMPLEMENTATION / DATABASE ENFORCEMENT
```

Ketiganya harus konsisten.

Domain tidak boleh bergantung pada:

- HTTP
- HTML
- HTMX
- PostgreSQL
- repository implementation
- session
- cookie
- presentation layer

Service bertugas mengorkestrasi use case dan authorization.

Database menjadi enforcement tambahan untuk invariant yang membutuhkan concurrency safety atau relational constraint.

---

# 3. Domain Inventory

Domain entity yang diaudit:

1. User
2. AcademicYear
3. Class
4. Student
5. StudentClassHistory
6. TeacherClassAssignment
7. ClassTransferRequest
8. SavingsAccount
9. Transaction
10. SavingsSettlement
11. AuditLog

Value object / supporting domain model:

1. Money
2. Balance
3. DateRange
4. Session
5. Actor-related types

---

# 4. USER

## 4.1 Entity

`domain.User`

Field utama:

```text
ID
Username
PasswordHash
Role
CreatedAt
UpdatedAt
```

## 4.2 Domain validation

`NewUser()` memastikan:

- username tidak kosong
- password hash tidak kosong
- role valid

Role yang tersedia:

```text
ADMIN
WALI_KELAS
SISWA
```

## 4.3 State transition

User tidak memiliki state transition domain khusus.

Role dapat diubah melalui service update.

## 4.4 Audit result

Status:

```text
MATCH
```

Tidak ditemukan konflik antara role domain dan database.

Database juga membatasi role:

```text
ADMIN
WALI_KELAS
SISWA
```

## 4.5 Catatan

Validasi password hash pada constructor masih bersifat:

```text
tidak kosong
```

Validitas format Argon2id merupakan concern security/service, bukan domain user constructor.

Status:

```text
MATCH
```

---

# 5. ACADEMIC YEAR

## 5.1 Entity

`domain.AcademicYear`

Field:

```text
ID
Name
StartDate
EndDate
Status
CreatedAt
UpdatedAt
```

Status:

```text
OPEN
CLOSING
CLOSED
```

## 5.2 Domain invariants

`NewAcademicYear()` memastikan:

```text
StartDate < EndDate
Name != empty
```

## 5.3 State machine

State transition domain:

```text
OPEN
  ↓ StartClosing()
CLOSING
  ↓ Close()
CLOSED
  ↓ Reopen()
OPEN
```

Transition lain ditolak.

## 5.4 Database enforcement

Database juga memiliki:

```text
start_date < end_date
```

dan status terbatas pada:

```text
OPEN
CLOSING
CLOSED
```

Selain itu terdapat unique partial index:

```text
only one OPEN academic year
```

## 5.5 Audit result

Status:

```text
MATCH
```

## 5.6 Catatan

Database saat ini belum memiliki constraint yang mencegah dua academic year memiliki periode yang saling overlap.

Domain juga belum memiliki invariant overlap antar academic year.

Temuan ini bukan domain entity invariant sederhana karena membutuhkan keputusan business contract mengenai apakah periode academic year boleh overlap.

Status:

```text
GAP - CONTRACT DECISION
```

ID:

```text
GAP-AY-01
```

Keputusan ditunda ke:

```text
FASE 16.2 Database Contract Audit
```

dan harus dipastikan kontraknya terlebih dahulu sebelum migration diubah.

---

# 6. CLASS

## 6.1 Entity

`domain.Class`

Field:

```text
ID
Name
Level
CreatedAt
UpdatedAt
```

## 6.2 Domain invariants

`NewClass()` memastikan:

```text
Name != empty
Level >= 1
Level <= 12
```

## 6.3 Database enforcement

Database memiliki:

```text
level BETWEEN 1 AND 12
```

dan:

```text
UNIQUE(name, level)
```

## 6.4 Audit result

Status:

```text
MATCH
```

---

# 7. STUDENT

## 7.1 Entity

`domain.Student`

Field:

```text
ID
UserID
NIS
NISN
Name
Status
CreatedAt
UpdatedAt
```

Status:

```text
ACTIVE
LEFT
```

## 7.2 Domain invariants

`NewStudent()` memastikan:

```text
Name != empty
NIS tidak nil
NIS tidak kosong
NISN tidak nil
NISN tidak kosong
```

Student baru selalu:

```text
ACTIVE
```

## 7.3 State transition

Student memiliki:

```text
ACTIVE → LEFT
```

melalui:

```text
Leave()
```

Transition:

```text
LEFT → ACTIVE
```

tidak tersedia.

## 7.4 Database enforcement

Migration `000003_require_student_identity` menjadikan:

```text
NIS NOT NULL
NISN NOT NULL
```

Database juga memiliki unique constraint untuk:

```text
user_id
nis
nisn
```

## 7.5 Audit result

Status:

```text
MATCH
```

Kontrak bahwa NIS dan NISN wajib telah tercermin baik di domain maupun database.

---

# 8. STUDENT CLASS HISTORY

## 8.1 Entity

`domain.StudentClassHistory`

Field:

```text
ID
StudentID
AcademicYearID
ClassID
Period
CreatedAt
UpdatedAt
```

## 8.2 Domain invariants

Constructor memastikan:

```text
StudentID != nil
AcademicYearID != nil
ClassID != nil
```

Period menggunakan:

```text
DateRange
```

yang memastikan:

```text
From < To
```

jika `To` tersedia.

## 8.3 Business contract

Kontrak:

```text
Siswa tidak boleh pindah kelas di tengah tahun
```

kemudian sistem memiliki mekanisme khusus:

```text
ClassTransferRequest
```

yang memungkinkan perpindahan kelas melalui approval.

Dengan demikian kontrak yang lebih tepat adalah:

```text
Tidak boleh ada perpindahan kelas langsung.

Perpindahan kelas hanya melalui
ClassTransferRequest → approval → history transition.
```

## 8.4 Database enforcement

Database memiliki exclusion constraint:

```text
student_id
academic_year_id
validity
```

yang mencegah period overlap untuk student + academic year.

## 8.5 Service enforcement

Class transfer approval:

1. mengambil current history
2. memvalidasi effective date
3. menutup history lama
4. membuat history baru
5. melakukan semuanya dalam transaction

## 8.6 Temuan

Domain entity sendiri tidak membatasi:

```text
satu history sepanjang academic year
```

karena model memang mendukung beberapa history non-overlapping untuk mekanisme transfer.

Ini bukan bug.

Status:

```text
MATCH
```

## 8.7 GAP initial assignment

Saat Student dibuat, `StudentService.Create()` belum secara eksplisit membuat:

```text
StudentClassHistory
```

Artinya lifecycle:

```text
Create Student
    ↓
Initial Class Assignment
```

belum diwujudkan sebagai use case yang eksplisit dalam StudentService.

Ini tidak merusak domain entity, tetapi merupakan gap implementasi lifecycle.

Status:

```text
GAP - IMPLEMENTATION
```

ID:

```text
GAP-STUDENT-CLASS-01
```

Owner:

```text
FASE 19 — Student Management
```

Tidak diperbaiki pada FASE 16.1.

---

# 9. TEACHER CLASS ASSIGNMENT

## 9.1 Entity

`domain.TeacherClassAssignment`

Field:

```text
ID
UserID
AcademicYearID
ClassID
Period
CreatedAt
UpdatedAt
```

## 9.2 Domain invariants

Constructor memastikan seluruh ID tidak nil.

Period menggunakan `DateRange`.

## 9.3 Database enforcement

Database mencegah overlap assignment untuk:

```text
academic_year_id
class_id
validity
```

## 9.4 Service contract

Service membedakan:

```text
TeacherClassAssignment
```

dari:

```text
StudentClassHistory
```

Assignment guru merupakan scope wali kelas.

## 9.5 Audit result

Status:

```text
MATCH
```

## 9.6 Catatan

Tidak ditemukan invariant domain tambahan yang harus ditambahkan pada entity.

---

# 10. CLASS TRANSFER REQUEST

## 10.1 Entity

`domain.ClassTransferRequest`

Field:

```text
ID
StudentID
AcademicYearID
FromClassID
ToClassID
RequestedBy
Status
ReviewedBy
ReviewedAt
RejectionReason
CreatedAt
UpdatedAt
```

Status:

```text
PENDING
APPROVED
REJECTED
```

## 10.2 Domain invariants

Constructor memastikan:

```text
StudentID != nil
AcademicYearID != nil
FromClassID != nil
ToClassID != nil
RequestedBy != nil
FromClassID != ToClassID
```

Initial state:

```text
PENDING
```

## 10.3 State machine

```text
PENDING
   ├── Approve() → APPROVED
   └── Reject()  → REJECTED
```

State terminal tidak dapat diproses kembali.

## 10.4 Approval contract

Approval membutuhkan:

```text
reviewedBy
reviewedAt
```

## 10.5 Rejection contract

Rejection membutuhkan:

```text
reviewedBy
reviewedAt
reason
```

Reason tidak boleh kosong.

## 10.6 Database enforcement

Database memiliki constraint untuk:

```text
from_class_id <> to_class_id
```

status valid.

Database juga memastikan:

```text
PENDING:
    reviewed_by IS NULL
    reviewed_at IS NULL

APPROVED/REJECTED:
    reviewed_by IS NOT NULL
    reviewed_at IS NOT NULL
```

## 10.7 Audit result

Status:

```text
MATCH
```

---

# 11. SAVINGS ACCOUNT

## 11.1 Entity

`domain.SavingsAccount`

Field:

```text
ID
StudentID
AcademicYearID
Status
SettledAt
CreatedAt
UpdatedAt
```

Status:

```text
OPEN
SETTLED
```

## 11.2 Domain invariants

Constructor memastikan:

```text
StudentID != nil
AcademicYearID != nil
```

Initial state:

```text
OPEN
```

## 11.3 State machine

```text
OPEN
  ↓ Settle()
SETTLED
  ↓ Reopen()
OPEN
```

## 11.4 Database enforcement

Unique:

```text
(student_id, academic_year_id)
```

Status:

```text
OPEN / SETTLED
```

`settled_at` konsisten dengan status.

## 11.5 Audit result

Status:

```text
MATCH
```

---

# 12. TRANSACTION

## 12.1 Entity

`domain.Transaction`

Field:

```text
ID
SavingsAccountID
Type
Amount
TransactionDate
Status
CreatedBy
CreatedAt
UpdatedAt
```

Type:

```text
DEPOSIT
WITHDRAWAL
```

Status:

```text
ACTIVE
CANCELLED
```

## 12.2 Domain invariants

Constructor memastikan:

```text
SavingsAccountID != nil
CreatedBy != nil
Type valid
Amount > 0
```

Transaction baru:

```text
ACTIVE
```

## 12.3 Cancellation state

```text
ACTIVE → CANCELLED
```

Cancellation kedua ditolak.

## 12.4 Balance invariant

`CalculateBalance()`:

```text
DEPOSIT    → +amount
WITHDRAWAL → -amount
```

Transaction `CANCELLED` tidak dihitung.

Balance tidak boleh negatif.

## 12.5 Database enforcement

Database memastikan:

```text
amount > 0
```

dan:

```text
amount = trunc(amount, 0)
```

Domain menggunakan:

```text
Money int64
```

sehingga fractional money tidak didukung domain.

## 12.6 Critical contract gap

Domain constructor tidak memvalidasi:

```text
TransactionDate
```

terhadap:

```text
AcademicYear.StartDate
AcademicYear.EndDate
```

Service transaction saat ini juga belum menunjukkan enforcement eksplisit:

```text
academic_year.start_date
<= transaction_date
< academic_year.end_date
```

Kontrak tanggal transaksi perlu dipastikan sebelum transaction engine dianggap final.

Status:

```text
GAP - IMPLEMENTATION
```

ID:

```text
GAP-TRANSACTION-DATE-01
```

Owner:

```text
FASE 21 — Transaction Engine
```

Kontrak target:

```text
start_date <= transaction_date < end_date
```

Perlu dipastikan apakah boundary ini merupakan business contract final pada FASE 16.5 sebelum implementation.

---

# 13. SAVINGS SETTLEMENT

## 13.1 Entity

`domain.SavingsSettlement`

Field:

```text
ID
SavingsAccountID
Type
Amount
ExecutedBy
ExecutedAt
Status
ReplacesSettlementID
```

Type:

```text
REGULAR_YEAR_END
STUDENT_LEAVING
```

Status:

```text
COMPLETED
SUPERSEDED
```

## 13.2 Domain invariants

Constructor memastikan:

```text
SavingsAccountID != nil
ExecutedBy != nil
Type valid
Amount >= 0
```

## 13.3 State machine

```text
COMPLETED
    ↓ Supersede()
SUPERSEDED
```

## 13.4 Settlement contract

Year-end settlement:

```text
AcademicYear harus CLOSING
SavingsAccount harus OPEN
```

Student leaving settlement:

```text
Student harus LEFT
SavingsAccount harus OPEN
```

## 13.5 Reopen

Current implementation:

```text
SUPERSEDE settlement
+
REOPEN savings account
```

dilakukan oleh:

```text
ADMIN
```

## 13.6 Contract observation

Tidak ditemukan invariant domain yang bertentangan.

Namun workflow approval untuk reopen oleh WALI_KELAS belum tersedia.

Status:

```text
DEFERRED
```

ID:

```text
GAP-SETTLEMENT-REOPEN-01
```

Alasan:

Reopen workflow membutuhkan keputusan business process terlebih dahulu.

Tidak boleh ditambahkan hanya berdasarkan asumsi.

---

# 14. MONEY

## 14.1 Value object

```text
type Money int64
```

Operasi:

```text
IsNegative()
IsZero()
IsPositive()
Add()
Subtract()
```

## 14.2 Contract

Money tidak boleh negatif.

Transaction amount harus positif.

Settlement amount boleh:

```text
0
```

## 14.3 Audit result

Status:

```text
MATCH
```

## 14.4 Catatan

Database menyimpan NUMERIC(15,2), tetapi migration `000002` memaksa:

```text
amount = trunc(amount, 0)
```

Domain menggunakan `int64`.

Dengan demikian kontrak aplikasi saat ini secara efektif adalah:

```text
whole-unit money
```

bukan fractional decimal money.

Status:

```text
MATCH
```

dengan catatan implementasi database.

---

# 15. BALANCE

## 15.1 Contract

Balance:

```text
>= 0
```

Perhitungan hanya menggunakan transaction:

```text
ACTIVE
```

Transaction:

```text
CANCELLED
```

tidak ikut dihitung.

## 15.2 Withdrawal

Withdrawal tidak boleh menghasilkan balance negatif.

## 15.3 Audit result

Status:

```text
MATCH
```

---

# 16. DATE RANGE

## 16.1 Contract

`DateRange`:

```text
From < To
```

jika `To` tersedia.

Jika `To == nil`:

```text
open-ended range
```

`Contains()` menggunakan interval:

```text
[From, To)
```

## 16.2 Audit result

Status:

```text
MATCH
```

## 16.3 Relevance

Kontrak ini konsisten dengan:

- StudentClassHistory
- TeacherClassAssignment
- ClassTransfer effective date

---

# 17. AUDIT LOG

## 17.1 Entity

`domain.AuditLog`

Field:

```text
ID
ActorUserID
Action
EntityType
EntityID
OccurredAt
BeforeData
AfterData
```

## 17.2 Entity type

Domain membatasi entity type kepada:

```text
USER
ACADEMIC_YEAR
CLASS
STUDENT
STUDENT_CLASS_HISTORY
TEACHER_CLASS_ASSIGNMENT
CLASS_TRANSFER_REQUEST
SAVINGS_ACCOUNT
TRANSACTION
SAVINGS_SETTLEMENT
```

## 17.3 Data validation

`BeforeData` dan `AfterData`:

```text
nil
atau valid JSON
```

## 17.4 Audit result

Entity model:

```text
MATCH
```

Coverage terhadap seluruh mutation bukan tanggung jawab constructor AuditLog.

Coverage mutation diaudit lebih lanjut pada:

```text
FASE 16.6 — Audit Contract Audit
```

---

# 18. DOMAIN STATE MACHINE SUMMARY

## Academic Year

```text
OPEN → CLOSING → CLOSED
 ↑                 │
 └─────────────────┘
```

Reopen hanya dari CLOSED.

Status:

```text
MATCH
```

## Student

```text
ACTIVE → LEFT
```

Status:

```text
MATCH
```

## Savings Account

```text
OPEN → SETTLED → OPEN
```

Status:

```text
MATCH
```

## Transaction

```text
ACTIVE → CANCELLED
```

Status:

```text
MATCH
```

## Class Transfer Request

```text
             ┌→ APPROVED
PENDING ─────┤
             └→ REJECTED
```

Status:

```text
MATCH
```

## Savings Settlement

```text
COMPLETED → SUPERSEDED
```

Status:

```text
MATCH
```

---

# 19. DOMAIN CONTRACT AUDIT MATRIX

| ID | Area | Result | Owner / Next Phase |
|---|---|---|---|
| GAP-AY-01 | Academic year overlapping period | GAP - CONTRACT DECISION | FASE 16.2 |
| GAP-STUDENT-CLASS-01 | Initial student class assignment | GAP - IMPLEMENTATION | FASE 19 |
| GAP-TRANSACTION-DATE-01 | Transaction date inside academic year | GAP - IMPLEMENTATION | FASE 21 |
| GAP-SETTLEMENT-REOPEN-01 | Reopen approval workflow | DEFERRED | Contract decision first |
| USER-01 | User role/domain validation | MATCH | — |
| CLASS-01 | Class level/name | MATCH | — |
| STUDENT-01 | Student identity | MATCH | — |
| STUDENT-02 | Student lifecycle | MATCH | — |
| CLASS-HISTORY-01 | Non-overlapping class history | MATCH | — |
| TEACHER-01 | Teacher assignment | MATCH | — |
| TRANSFER-01 | Transfer state machine | MATCH | — |
| ACCOUNT-01 | Account lifecycle | MATCH | — |
| TRANSACTION-01 | Transaction amount/type/status | MATCH | — |
| BALANCE-01 | Non-negative balance | MATCH | — |
| SETTLEMENT-01 | Settlement lifecycle | MATCH | — |
| MONEY-01 | Whole-unit money | MATCH | — |
| DATE-RANGE-01 | Half-open period | MATCH | — |
| AUDIT-01 | Audit entity model | MATCH | FASE 16.6 coverage audit |

---

# 20. Temuan yang TIDAK BOLEH DIPERBAIKI PADA FASE 16.1

Temuan berikut sengaja tidak langsung diubah:

## 20.1 Academic year overlap

Belum diputuskan apakah:

```text
Academic year periods
```

boleh overlap.

Jangan langsung menambahkan PostgreSQL exclusion constraint.

---

## 20.2 Initial student class assignment

Initial class assignment merupakan lifecycle/use-case concern.

Jangan menambahkan history secara otomatis ke `NewStudent()`.

Domain constructor Student tidak boleh mengetahui:

```text
AcademicYear
Class
```

---

## 20.3 Transaction date boundary

Jangan memasukkan AcademicYear ke constructor Transaction hanya untuk memaksa validasi.

Validasi lintas aggregate/entity lebih tepat berada pada service/use case.

Contract perlu dikunci terlebih dahulu.

---

## 20.4 Settlement reopen workflow

Jangan membuat workflow:

```text
WALI_KELAS request
→ ADMIN approve
```

sebelum business contract memutuskan bahwa workflow tersebut memang diperlukan.

---

# 21. Kesimpulan FASE 16.1

Domain model saat ini secara umum konsisten dengan kontrak aplikasi.

Mayoritas invariant utama sudah mempunyai enforcement pada:

```text
Domain
+
Service
+
Database
```

Temuan utama:

```text
1. Academic year overlap belum diputuskan.
2. Initial student class assignment belum menjadi use case eksplisit.
3. Transaction date boundary perlu dikunci dan diimplementasikan.
4. Settlement reopen approval workflow masih deferred.
```

Tidak ditemukan conflict fatal pada entity state machine utama.

Tidak ada perubahan source code yang dilakukan pada FASE 16.1.

---

# 22. Status

```text
FASE 16.1 DOMAIN CONTRACT AUDIT

Status: COMPLETE

Domain audit:
    COMPLETE

Source modification:
    NONE

Contract changes:
    NONE

Known gaps:
    4

Fatal conflicts:
    0

Next:
    FASE 16.2 Database Contract Audit
```

---

# 23. Definition of Done

FASE 16.1 dianggap selesai apabila:

- [x] Semua domain entity utama diaudit.
- [x] Value object utama diaudit.
- [x] State transition utama diaudit.
- [x] Domain invariant dicatat.
- [x] Database enforcement yang relevan dibandingkan.
- [x] Gap implementation dicatat.
- [x] Gap contract decision dicatat.
- [x] Deferred decision dicatat.
- [x] Tidak ada source code yang diubah.
- [x] Temuan diberi owner/phase.
- [x] Tidak ada conflict fatal yang belum teridentifikasi.

