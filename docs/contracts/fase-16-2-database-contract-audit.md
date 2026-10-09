# FASE 16.2 — DATABASE CONTRACT AUDIT

Status: AUDIT COMPLETE
Phase: 16.2
Parent: FASE 16 — CONTRACT HARDENING
Scope: PostgreSQL schema, constraints, foreign keys, indexes, range enforcement, migration consistency
Source baseline: commit `1a680c8`
Audit basis: source snapshot `tabungan-siswa.tar(20261008-034155).gz`

---

# 1. Tujuan

FASE 16.2 melakukan audit terhadap database contract aplikasi Tabungan Siswa.

Audit membandingkan:

1. Domain contract.
2. Business contract yang telah ditetapkan.
3. PostgreSQL schema.
4. CHECK constraint.
5. UNIQUE constraint/index.
6. EXCLUDE constraint.
7. FOREIGN KEY.
8. NULL/NOT NULL contract.
9. State consistency.
10. Index yang mendukung access pattern.
11. Migration ordering.
12. Migration up/down consistency.
13. Database enforcement terhadap invariant yang membutuhkan concurrency safety.

FASE 16.2 tidak mengubah migration.

Temuan hanya dicatat dan diklasifikasikan.

---

# 2. Classification

Setiap temuan menggunakan salah satu status:

```text
MATCH
GAP - IMPLEMENTATION
GAP - CONTRACT DECISION
CONFLICT
DEFERRED
```

Interpretasi:

## MATCH

Database sudah konsisten dengan kontrak.

## GAP - IMPLEMENTATION

Kontrak sudah jelas, tetapi database belum memberikan enforcement yang diperlukan.

## GAP - CONTRACT DECISION

Belum boleh mengubah database karena kontraknya sendiri masih membutuhkan keputusan.

## CONFLICT

Database secara langsung bertentangan dengan kontrak yang telah dikunci.

## DEFERRED

Temuan valid tetapi sengaja ditunda ke fase yang memiliki ownership lebih tepat.

---

# 3. Migration Inventory

Migration saat ini:

```text
000001_init_schema
000002_align_domain_contract
000003_require_student_identity
000004_sessions
```

Urutan dependency:

```text
000001
   ↓
000002
   ↓
000003
   ↓
000004
```

## 3.1 000001

Membuat:

```text
users
academic_years
classes
students
student_class_histories
teacher_class_assignments
class_transfer_requests
savings_accounts
transactions
savings_settlements
audit_logs
```

dan:

```text
pgcrypto
btree_gist
```

## 3.2 000002

Menyelaraskan:

```text
updated_at
rejection_reason
integer money constraint
```

## 3.3 000003

Mewajibkan:

```text
students.nis
students.nisn
```

## 3.4 000004

Membuat:

```text
sessions
```

---

# 4. EXTENSIONS

Migration `000001` membuat:

```sql
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS btree_gist;
```

## 4.1 pgcrypto

Digunakan untuk:

```sql
gen_random_uuid()
```

Status:

```text
MATCH
```

## 4.2 btree_gist

Diperlukan oleh:

```text
EXCLUDE USING GIST
```

pada range-based constraint.

Status:

```text
MATCH
```

## 4.3 Down migration

Extension dihapus setelah seluruh table yang menggunakannya dihapus.

Urutan:

```text
tables
  ↓
extensions
```

Status:

```text
MATCH
```

---

# 5. USERS

Schema:

```text
id UUID PRIMARY KEY
username TEXT NOT NULL UNIQUE
password_hash TEXT NOT NULL
role TEXT NOT NULL
created_at TIMESTAMPTZ NOT NULL
updated_at TIMESTAMPTZ NOT NULL
```

## 5.1 ID

```text
PRIMARY KEY
```

Status:

```text
MATCH
```

## 5.2 Username

```text
NOT NULL
UNIQUE
```

Sesuai contract username sebagai identity login.

Status:

```text
MATCH
```

## 5.3 Password hash

```text
NOT NULL
```

Database tidak memvalidasi format Argon2id.

Ini benar karena format password hash merupakan security/application concern.

Status:

```text
MATCH
```

## 5.4 Role

Constraint:

```sql
CHECK (
    role IN (
        'ADMIN',
        'WALI_KELAS',
        'SISWA'
    )
)
```

Sama dengan domain.

Status:

```text
MATCH
```

## 5.5 Timestamps

```text
created_at DEFAULT NOW()
updated_at DEFAULT NOW()
```

Database tidak otomatis memperbarui `updated_at`.

Application/service bertanggung jawab memperbaruinya.

Status:

```text
MATCH
```

Catatan:

Ini merupakan application-managed timestamp contract, bukan database-trigger contract.

---

# 6. ACADEMIC YEARS

Schema:

```text
id UUID PRIMARY KEY
name TEXT NOT NULL
start_date DATE NOT NULL
end_date DATE NOT NULL
status TEXT NOT NULL
created_at TIMESTAMPTZ NOT NULL
updated_at TIMESTAMPTZ NOT NULL
```

## 6.1 Date order

Constraint:

```sql
CHECK (start_date < end_date)
```

Domain juga menggunakan:

```text
StartDate < EndDate
```

Status:

```text
MATCH
```

## 6.2 Status

Constraint:

```text
OPEN
CLOSING
CLOSED
```

Sesuai domain.

Status:

```text
MATCH
```

## 6.3 One OPEN academic year

Partial unique index:

```sql
CREATE UNIQUE INDEX academic_years_one_open_idx
    ON academic_years (status)
    WHERE status = 'OPEN';
```

Contract:

```text
maximum one OPEN academic year
```

Status:

```text
MATCH
```

## 6.4 Academic year overlap

Saat ini database tidak memiliki constraint yang mencegah:

```text
AY 2026/2027
2026-07-01 → 2027-06-30

AY 2027/2028
2027-01-01 → 2027-12-31
```

Contoh tersebut dapat overlap walaupun keduanya tidak harus berstatus OPEN.

Domain juga belum mendefinisikan overlap sebagai invariant entity-level.

Status:

```text
GAP - CONTRACT DECISION
```

ID:

```text
GAP-DB-AY-01
```

Keputusan diperlukan terlebih dahulu:

```text
Apakah seluruh academic year period harus saling non-overlapping?
```

Jangan menambahkan exclusion constraint sebelum keputusan tersebut dikunci.

Owner:

```text
FASE 16.4 / 16.7
```

---

# 7. CLASSES

Schema:

```text
id UUID PRIMARY KEY
name TEXT NOT NULL
level SMALLINT NOT NULL
created_at TIMESTAMPTZ NOT NULL
updated_at TIMESTAMPTZ NOT NULL
```

## 7.1 Level

Constraint:

```sql
CHECK (level BETWEEN 1 AND 12)
```

Domain:

```text
1 <= level <= 12
```

Status:

```text
MATCH
```

## 7.2 Name + level

Unique:

```text
(name, level)
```

Sesuai contract.

Status:

```text
MATCH
```

## 7.3 Name normalization

Database tidak melakukan:

```text
TRIM
LOWER
case folding
```

Domain hanya memastikan nama tidak kosong.

Kontrak saat ini tidak menyatakan bahwa:

```text
"7A"
```

dan:

```text
" 7A "
```

harus dianggap identik.

Status:

```text
MATCH
```

Tidak perlu menambah functional unique index tanpa business decision.

---

# 8. STUDENTS

Schema awal:

```text
id UUID PRIMARY KEY
user_id UUID UNIQUE REFERENCES users(id)
nis TEXT UNIQUE
nisn TEXT UNIQUE
name TEXT NOT NULL
status TEXT NOT NULL
created_at TIMESTAMPTZ NOT NULL
updated_at TIMESTAMPTZ NOT NULL
```

Migration `000003` kemudian membuat:

```text
nis NOT NULL
nisn NOT NULL
```

## 8.1 Student ID

```text
PRIMARY KEY
```

Status:

```text
MATCH
```

## 8.2 User ID

```text
UNIQUE
REFERENCES users(id)
```

Tetapi nullable.

Artinya:

```text
student dapat tidak memiliki user account
```

Ini konsisten dengan domain:

```text
UserID *uuid.UUID
```

dan dengan kontrak bahwa student login tidak menjadi identity utama aplikasi.

Status:

```text
MATCH
```

## 8.3 NIS

Final state:

```text
NOT NULL
UNIQUE
```

Sesuai contract.

Status:

```text
MATCH
```

## 8.4 NISN

Final state:

```text
NOT NULL
UNIQUE
```

Sesuai contract.

Status:

```text
MATCH
```

## 8.5 Student status

Constraint:

```text
ACTIVE
LEFT
```

Sesuai domain.

Status:

```text
MATCH
```

## 8.6 Student name

```text
NOT NULL
```

Domain juga melarang nama kosong.

Database tidak dapat mencegah:

```text
''
'   '
```

Kontrak database saat ini hanya:

```text
NOT NULL
```

dan domain menangani semantic validation.

Status:

```text
MATCH
```

---

# 9. STUDENT CLASS HISTORIES

Schema:

```text
id UUID PRIMARY KEY
student_id UUID NOT NULL REFERENCES students(id)
academic_year_id UUID NOT NULL REFERENCES academic_years(id)
class_id UUID NOT NULL REFERENCES classes(id)
validity DATERANGE NOT NULL
created_at TIMESTAMPTZ NOT NULL
updated_at TIMESTAMPTZ NOT NULL
```

## 9.1 Foreign keys

Semua parent reference diwajibkan.

Status:

```text
MATCH
```

## 9.2 Empty range

Constraint:

```sql
CHECK (NOT isempty(validity))
```

Sesuai kebutuhan period.

Status:

```text
MATCH
```

## 9.3 No overlap

Exclusion:

```sql
EXCLUDE USING GIST (
    student_id WITH =,
    academic_year_id WITH =,
    validity WITH &&
)
```

Artinya satu student tidak dapat memiliki dua history yang overlap dalam academic year yang sama.

Ini merupakan concurrency-safe database enforcement.

Status:

```text
MATCH
```

## 9.4 Period inside academic year

Database belum memastikan:

```text
lower(validity) >= academic_year.start_date
```

dan:

```text
upper(validity) <= academic_year.end_date
```

Service transfer secara eksplisit memvalidasi effective date terhadap academic year.

Namun repository database secara langsung masih dapat menerima range yang berada di luar academic year.

Status:

```text
GAP - CONTRACT DECISION
```

ID:

```text
GAP-DB-HISTORY-01
```

Alasan tidak langsung mengubah:

PostgreSQL CHECK constraint tidak dapat secara sederhana melakukan lookup ke table `academic_years`.

Alternatif enforcement:

```text
service-only
trigger
schema redesign
```

masing-masing mempunyai konsekuensi.

Kontrak harus diputuskan terlebih dahulu.

---

# 10. TEACHER CLASS ASSIGNMENTS

Schema:

```text
id UUID PRIMARY KEY
user_id UUID NOT NULL REFERENCES users(id)
academic_year_id UUID NOT NULL REFERENCES academic_years(id)
class_id UUID NOT NULL REFERENCES classes(id)
validity DATERANGE NOT NULL
created_at TIMESTAMPTZ NOT NULL
updated_at TIMESTAMPTZ NOT NULL
```

## 10.1 Teacher reference

FK ke:

```text
users
```

Service memastikan user mempunyai role:

```text
WALI_KELAS
```

Database hanya memastikan user exists.

Status:

```text
MATCH
```

Role enforcement memang bukan foreign-key concern.

## 10.2 Class/year reference

```text
NOT NULL REFERENCES
```

Status:

```text
MATCH
```

## 10.3 Empty range

```text
CHECK (NOT isempty(validity))
```

Status:

```text
MATCH
```

## 10.4 Overlap

Constraint:

```sql
EXCLUDE USING GIST (
    academic_year_id WITH =,
    class_id WITH =,
    validity WITH &&
)
```

Kontrak:

```text
satu class tidak memiliki dua teacher assignment
yang overlap pada academic year yang sama.
```

Status:

```text
MATCH
```

## 10.5 Period inside academic year

Database tidak secara langsung memastikan validity berada di dalam academic year.

Service assignment saat ini membuat period:

```text
academic_year.start_date
→ academic_year.end_date
```

sehingga use case normal memenuhi contract.

Database belum menjadi final enforcement untuk arbitrary repository writes.

Status:

```text
DEFERRED
```

Temuan berkaitan dengan:

```text
GAP-DB-HISTORY-01
```

dan dapat diselesaikan sebagai satu keputusan range policy.

---

# 11. CLASS TRANSFER REQUESTS

Schema:

```text
id UUID PRIMARY KEY
student_id UUID NOT NULL REFERENCES students(id)
academic_year_id UUID NOT NULL REFERENCES academic_years(id)
from_class_id UUID NOT NULL REFERENCES classes(id)
to_class_id UUID NOT NULL REFERENCES classes(id)
requested_by UUID NOT NULL REFERENCES users(id)
requested_at TIMESTAMPTZ NOT NULL
status TEXT NOT NULL
reviewed_by UUID REFERENCES users(id)
reviewed_at TIMESTAMPTZ
updated_at TIMESTAMPTZ NOT NULL
rejection_reason TEXT
```

## 11.1 Different source/target class

Constraint:

```sql
CHECK (from_class_id <> to_class_id)
```

Sesuai domain.

Status:

```text
MATCH
```

## 11.2 Status

Constraint:

```text
PENDING
APPROVED
REJECTED
```

Sesuai domain.

Status:

```text
MATCH
```

## 11.3 Review consistency

Constraint:

```text
PENDING
    → reviewed_by NULL
    → reviewed_at NULL

APPROVED / REJECTED
    → reviewed_by NOT NULL
    → reviewed_at NOT NULL
```

Sesuai domain state transition.

Status:

```text
MATCH
```

## 11.4 Rejection reason

Migration `000002` menambahkan:

```text
rejection_reason TEXT
```

Database belum memiliki constraint:

```text
APPROVED → rejection_reason NULL
REJECTED → rejection_reason NOT NULL
PENDING  → rejection_reason NULL
```

Domain:

```text
Approve()
    → RejectionReason = nil

Reject()
    → RejectionReason wajib tidak kosong
```

Database saat ini hanya sebagian mencerminkan invariant tersebut.

Status:

```text
GAP - IMPLEMENTATION
```

ID:

```text
GAP-DB-TRANSFER-01
```

Namun keputusan implementasi perlu mempertimbangkan apakah whitespace-only rejection reason harus ditolak database.

Domain sudah melakukan:

```text
strings.TrimSpace(reason) == ""
```

Database CHECK hanya dapat meniru secara sederhana menggunakan:

```text
NULLIF(BTRIM(rejection_reason), '')
```

Perubahan ini ditunda sampai contract database final.

---

# 12. MULTIPLE PENDING TRANSFER REQUESTS

Database saat ini tidak memiliki unique constraint seperti:

```text
(student_id, academic_year_id)
WHERE status = 'PENDING'
```

Service juga tidak secara eksplisit menolak request kedua yang masih pending.

Akibatnya secara database dapat terdapat:

```text
Student A
Academic Year X
PENDING request 1

Student A
Academic Year X
PENDING request 2
```

## Contract status

Belum ada business contract eksplisit yang menyatakan:

```text
satu student hanya boleh memiliki satu pending transfer request.
```

Karena itu:

```text
GAP - CONTRACT DECISION
```

ID:

```text
GAP-DB-TRANSFER-02
```

Jangan menambahkan partial unique index sebelum kontrak diputuskan.

---

# 13. SAVINGS ACCOUNTS

Schema:

```text
id UUID PRIMARY KEY
student_id UUID NOT NULL REFERENCES students(id)
academic_year_id UUID NOT NULL REFERENCES academic_years(id)
status TEXT NOT NULL
settled_at TIMESTAMPTZ
created_at TIMESTAMPTZ NOT NULL
updated_at TIMESTAMPTZ NOT NULL
```

## 13.1 One account per student/year

Unique:

```text
(student_id, academic_year_id)
```

Sesuai contract.

Status:

```text
MATCH
```

## 13.2 Account status

Constraint:

```text
OPEN
SETTLED
```

Sesuai domain.

Status:

```text
MATCH
```

## 13.3 settled_at consistency

Constraint:

```text
OPEN
    → settled_at IS NULL

SETTLED
    → settled_at IS NOT NULL
```

Sesuai domain state.

Status:

```text
MATCH
```

---

# 14. TRANSACTIONS

Schema:

```text
id UUID PRIMARY KEY
savings_account_id UUID NOT NULL REFERENCES savings_accounts(id)
type TEXT NOT NULL
amount NUMERIC(15,2) NOT NULL
transaction_date DATE NOT NULL
status TEXT NOT NULL
created_by UUID NOT NULL REFERENCES users(id)
created_at TIMESTAMPTZ NOT NULL
updated_at TIMESTAMPTZ NOT NULL
```

## 14.1 Type

Constraint:

```text
DEPOSIT
WITHDRAWAL
```

Status:

```text
MATCH
```

## 14.2 Amount positive

Constraint:

```text
CHECK (amount > 0)
```

Domain:

```text
Money.IsPositive()
```

Status:

```text
MATCH
```

## 14.3 Whole-unit amount

Migration `000002` adds:

```sql
CHECK (amount = trunc(amount, 0))
```

Domain uses:

```text
Money int64
```

Status:

```text
MATCH
```

## 14.4 Status

Constraint:

```text
ACTIVE
CANCELLED
```

Sesuai domain.

Status:

```text
MATCH
```

## 14.5 Transaction date

Database hanya memastikan:

```text
DATE NOT NULL
```

Tidak memastikan:

```text
academic_year.start_date
<= transaction_date
< academic_year.end_date
```

Database juga tidak memiliki direct FK from transaction to academic_year.

Academic year diketahui melalui:

```text
transaction
→ savings_account
→ academic_year
```

Service contract saat ini juga belum mempunyai final database enforcement untuk boundary tersebut.

Status:

```text
GAP - IMPLEMENTATION
```

ID:

```text
GAP-DB-TRANSACTION-01
```

Owner:

```text
FASE 21 — Transaction Engine
```

Kontrak target dari FASE 16.1:

```text
start_date <= transaction_date < end_date
```

Namun sebelum migration enforcement dirancang, contract boundary harus dikunci pada financial contract audit.

---

# 15. TRANSACTION STATUS AND DELETION

Tidak terdapat:

```text
DELETE
```

state pada transaction.

Contract menggunakan:

```text
ACTIVE → CANCELLED
```

Database tidak memberikan soft-delete column terpisah.

Status:

```text
MATCH
```

Database FK mencegah account parent dihapus ketika transaction masih mereferensikannya.

---

# 16. SAVINGS SETTLEMENTS

Schema:

```text
id UUID PRIMARY KEY
savings_account_id UUID NOT NULL REFERENCES savings_accounts(id)
type TEXT NOT NULL
amount NUMERIC(15,2) NOT NULL
executed_by UUID NOT NULL REFERENCES users(id)
executed_at TIMESTAMPTZ NOT NULL
status TEXT NOT NULL
replaces_settlement_id UUID REFERENCES savings_settlements(id)
```

## 16.1 Type

Constraint:

```text
REGULAR_YEAR_END
STUDENT_LEAVING
```

Status:

```text
MATCH
```

## 16.2 Amount

Constraint:

```text
amount >= 0
```

Sesuai domain.

Status:

```text
MATCH
```

## 16.3 Whole-unit amount

Constraint dari `000002`:

```text
amount = trunc(amount, 0)
```

Sesuai Money.

Status:

```text
MATCH
```

## 16.4 Status

Constraint:

```text
COMPLETED
SUPERSEDED
```

Sesuai domain.

Status:

```text
MATCH
```

## 16.5 One completed settlement

Partial unique index:

```text
ONE COMPLETED settlement
per savings account
```

Sesuai service contract:

```text
account dapat mempunyai satu settlement aktif/current.
```

Status:

```text
MATCH
```

## 16.6 replaces_settlement_id

FK:

```text
savings_settlements(id)
```

Tetapi database belum memastikan:

```text
replaces_settlement_id
```

berasal dari:

```text
same savings_account_id
```

Database juga tidak mencegah:

```text
replaces_settlement_id = own id
```

Domain hanya memastikan UUID tidak nil.

Saat ini field tersebut tidak menjadi bagian utama workflow Reopen service.

Status:

```text
DEFERRED
```

ID:

```text
GAP-DB-SETTLEMENT-01
```

Tidak boleh ditambahkan constraint tambahan sebelum settlement lineage contract ditetapkan.

---

# 17. AUDIT LOGS

Schema:

```text
id UUID PRIMARY KEY
actor_user_id UUID REFERENCES users(id)
action TEXT NOT NULL
entity_type TEXT NOT NULL
entity_id UUID NOT NULL
occurred_at TIMESTAMPTZ NOT NULL
before_data JSONB
after_data JSONB
```

## 17.1 Entity type

CHECK membatasi entity type ke domain list:

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

Status:

```text
MATCH
```

## 17.2 Entity ID

```text
NOT NULL
```

Sesuai domain.

Status:

```text
MATCH
```

## 17.3 JSON data

Database menggunakan:

```text
JSONB
```

yang memberikan valid JSON enforcement.

Domain juga menggunakan:

```text
json.Valid()
```

Status:

```text
MATCH
```

## 17.4 Actor

`actor_user_id` nullable.

Domain juga menggunakan:

```text
*uuid.UUID
```

sehingga system audit tanpa actor masih dimungkinkan.

Tidak ada contract yang saat ini mewajibkan actor selalu ada.

Status:

```text
MATCH
```

---

# 18. FOREIGN KEY DELETE POLICY

Seluruh foreign key pada schema saat ini menggunakan default PostgreSQL:

```text
NO ACTION
```

Tidak ada:

```text
ON DELETE CASCADE
```

pada relationship domain utama.

Contoh:

```text
academic_years
    ↑
student_class_histories
```

dan:

```text
savings_accounts
    ↑
transactions
```

Parent tidak dapat dihapus ketika masih direferensikan.

Ini konsisten dengan contract bahwa historical/financial records tidak boleh hilang hanya karena parent entity dihapus.

Status:

```text
MATCH
```

## 18.1 Important distinction

`NO ACTION` bukan secara textual sama dengan:

```text
ON DELETE RESTRICT
```

tetapi pada foreign key non-deferrable yang digunakan schema ini, efek operasionalnya adalah parent deletion ditolak ketika masih memiliki child reference.

Contract tidak membutuhkan cascade deletion.

---

# 19. HISTORICAL DATA PROTECTION

Database tidak memiliki cascade delete terhadap:

```text
transactions
settlements
audit_logs
student_class_histories
```

Ini merupakan keputusan yang tepat untuk data historis.

Status:

```text
MATCH
```

---

# 20. SESSION TABLE

Migration `000004`:

```text
sessions
```

Schema:

```text
id UUID PRIMARY KEY
user_id UUID NOT NULL REFERENCES users(id)
token_hash TEXT NOT NULL UNIQUE
expires_at TIMESTAMPTZ NOT NULL
revoked_at TIMESTAMPTZ
created_at TIMESTAMPTZ NOT NULL
```

## 20.1 Token hash

```text
NOT NULL
UNIQUE
```

Sesuai session contract.

Status:

```text
MATCH
```

## 20.2 Expiration

```text
NOT NULL
```

Domain memastikan expiration future saat creation.

Database tidak melakukan:

```text
expires_at > created_at
```

Ini application/domain concern.

Status:

```text
MATCH
```

## 20.3 Revocation

```text
revoked_at nullable
```

Sesuai domain.

Status:

```text
MATCH
```

## 20.4 User reference

```text
user_id NOT NULL REFERENCES users(id)
```

Sesi selalu memiliki owner.

Status:

```text
MATCH
```

---

# 21. INDEX AUDIT

## 21.1 Users

```text
users.username UNIQUE
```

Mendukung lookup authentication.

Status:

```text
MATCH
```

## 21.2 Students

Unique indexes:

```text
user_id
nis
nisn
```

Mendukung identity lookup.

Status:

```text
MATCH
```

## 21.3 Student class history

```text
(student_id, academic_year_id)
```

Mendukung:

```text
ListByStudentAndAcademicYear
GetCurrentByStudentAndAcademicYear
```

Status:

```text
MATCH
```

## 21.4 Teacher assignments

```text
(user_id, academic_year_id)
```

mendukung teacher/year listing.

Status:

```text
MATCH
```

## 21.5 Class transfer requests

```text
(student_id, academic_year_id)
status
```

mendukung:

```text
ListByStudentAndAcademicYear
ListPending
```

Status:

```text
MATCH
```

## 21.6 Savings accounts

```text
student_id
academic_year_id
```

mendukung lookup.

Status:

```text
MATCH
```

## 21.7 Transactions

```text
(savings_account_id, transaction_date)
created_by
```

mendukung account transaction history dan date ordering/filtering.

Status:

```text
MATCH
```

## 21.8 Settlements

```text
savings_account_id
```

mendukung settlement lookup.

Status:

```text
MATCH
```

## 21.9 Audit logs

```text
(entity_type, entity_id)
(actor_user_id)
(occurred_at)
```

mendukung audit queries.

Status:

```text
MATCH
```

## 21.10 Sessions

```text
user_id
expires_at
```

mendukung session lookup/maintenance.

Status:

```text
MATCH
```

---

# 22. MIGRATION UP/DOWN AUDIT

## 22.1 000001

UP membuat:

```text
extensions
tables
indexes
```

DOWN menghapus:

```text
tables
extensions
```

dalam dependency order.

Status:

```text
MATCH
```

## 22.2 000002

UP:

```text
ADD updated_at
ADD rejection_reason
ADD integer amount constraints
```

DOWN:

```text
DROP integer amount constraints
DROP columns
```

Status:

```text
MATCH
```

## 22.3 000003

UP:

```text
nis NOT NULL
nisn NOT NULL
```

DOWN:

```text
nis nullable
nisn nullable
```

Status:

```text
MATCH
```

## 22.4 000004

UP:

```text
CREATE sessions
```

DOWN:

```text
DROP sessions
```

Status:

```text
MATCH
```

---

# 23. MIGRATION CONSISTENCY

Migration sequence menghasilkan final schema:

```text
users
academic_years
classes
students
student_class_histories
teacher_class_assignments
class_transfer_requests
savings_accounts
transactions
savings_settlements
audit_logs
sessions
```

Final domain fields yang memerlukan migration tambahan:

```text
StudentClassHistory.updated_at
TeacherClassAssignment.updated_at
ClassTransferRequest.updated_at
ClassTransferRequest.rejection_reason
Transaction integer amount
Settlement integer amount
Student NIS NOT NULL
Student NISN NOT NULL
Session
```

Seluruh field tersebut telah tercakup migration.

Status:

```text
MATCH
```

---

# 24. DATABASE CONTRACT MATRIX

| ID | Area | Finding | Status | Owner |
|---|---|---|---|---|
| DB-01 | Users role | Role CHECK matches domain | MATCH | — |
| DB-02 | Users username | Unique + NOT NULL | MATCH | — |
| DB-03 | Academic year date | start < end | MATCH | — |
| DB-04 | Academic year status | Valid status set | MATCH | — |
| DB-05 | Academic year OPEN | Maximum one OPEN | MATCH | — |
| GAP-DB-AY-01 | Academic year overlap | Period overlap undefined | GAP - CONTRACT DECISION | FASE 16.7 |
| DB-06 | Class level | 1..12 | MATCH | — |
| DB-07 | Class identity | name + level unique | MATCH | — |
| DB-08 | Student identity | NIS/NISN final NOT NULL + UNIQUE | MATCH | — |
| DB-09 | Student user | Nullable unique user reference | MATCH | — |
| DB-10 | Student status | ACTIVE/LEFT | MATCH | — |
| DB-11 | Student history empty range | rejected | MATCH | — |
| DB-12 | Student history overlap | exclusion constraint | MATCH | — |
| GAP-DB-HISTORY-01 | History period | Not constrained inside AY | GAP - CONTRACT DECISION | FASE 16.7 |
| DB-13 | Teacher assignment overlap | exclusion constraint | MATCH | — |
| DB-14 | Teacher role | Service-level enforcement | MATCH | — |
| DB-15 | Transfer classes | from != to | MATCH | — |
| DB-16 | Transfer status | Valid status set | MATCH | — |
| DB-17 | Transfer review fields | State consistency | MATCH | — |
| GAP-DB-TRANSFER-01 | Rejection reason | DB does not enforce state-specific reason | GAP - IMPLEMENTATION | FASE 22 |
| GAP-DB-TRANSFER-02 | Pending requests | Multiple pending requests allowed | GAP - CONTRACT DECISION | FASE 16.7 |
| DB-18 | Savings account identity | student + year unique | MATCH | — |
| DB-19 | Savings account status | Valid states | MATCH | — |
| DB-20 | settled_at consistency | CHECK | MATCH | — |
| DB-21 | Transaction amount | > 0 | MATCH | — |
| DB-22 | Transaction integer | whole-unit CHECK | MATCH | — |
| DB-23 | Transaction type | valid types | MATCH | — |
| DB-24 | Transaction status | valid states | MATCH | — |
| GAP-DB-TRANSACTION-01 | Transaction date | Not bounded to academic year | GAP - IMPLEMENTATION | FASE 21 |
| DB-25 | Settlement amount | >= 0 | MATCH | — |
| DB-26 | Settlement integer | whole-unit CHECK | MATCH | — |
| DB-27 | Settlement status | valid states | MATCH | — |
| DB-28 | One completed settlement | partial unique index | MATCH | — |
| GAP-DB-SETTLEMENT-01 | Settlement lineage | replaces ID not same-account constrained | DEFERRED | FASE 22 |
| DB-29 | Audit entity type | CHECK | MATCH | — |
| DB-30 | Audit JSON | JSONB | MATCH | — |
| DB-31 | Audit entity ID | NOT NULL | MATCH | — |
| DB-32 | Historical deletion | No cascade deletion | MATCH | — |
| DB-33 | Sessions token hash | NOT NULL + UNIQUE | MATCH | — |
| DB-34 | Session owner | NOT NULL FK | MATCH | — |
| DB-35 | Session expiry | NOT NULL | MATCH | — |
| DB-36 | Query indexes | Core access patterns covered | MATCH | — |
| DB-37 | Migration ordering | Dependency order valid | MATCH | — |
| DB-38 | Migration down | Reversible | MATCH | — |

---

# 25. CRITICAL FINDINGS

Database audit menghasilkan empat temuan yang perlu dibawa ke contract hardening.

## 25.1 GAP-DB-AY-01

```text
Academic year overlap
```

Belum diputuskan apakah period academic year harus globally non-overlapping.

Tidak ada migration yang diubah.

---

## 25.2 GAP-DB-HISTORY-01

```text
StudentClassHistory validity
```

belum database-enforced berada di dalam academic year.

Service normal sudah menggunakan academic year boundary pada transfer.

Perlu keputusan apakah database harus menjadi enforcement tambahan.

---

## 25.3 GAP-DB-TRANSFER-01

Database belum memastikan:

```text
REJECTED → rejection_reason wajib
APPROVED → rejection_reason NULL
PENDING  → rejection_reason NULL
```

Domain sudah mempunyai aturan tersebut.

Ini merupakan candidate database constraint.

Implementasi ditunda sampai contract review selesai.

---

## 25.4 GAP-DB-TRANSACTION-01

Transaction date belum dibatasi database terhadap academic year.

Ini konsisten dengan temuan domain:

```text
GAP-TRANSACTION-DATE-01
```

Owner implementasi:

```text
FASE 21
```

Contract financial boundary harus dikunci lebih dahulu.

---

# 26. TEMUAN YANG TIDAK BOLEH LANGSUNG DIUBAH

Pada fase ini jangan melakukan:

```text
ALTER TABLE academic_years ...
ALTER TABLE student_class_histories ...
ALTER TABLE class_transfer_requests ...
ALTER TABLE transactions ...
```

untuk memperbaiki temuan di atas.

Alasannya:

1. Beberapa merupakan contract decision.
2. Beberapa membutuhkan desain enforcement lintas table.
3. Beberapa merupakan ownership fase implementasi berikutnya.
4. Migration merupakan bagian dari database contract yang harus diubah hanya setelah contract final.

---

# 27. DOMAIN ↔ DATABASE ENFORCEMENT SUMMARY

## Strongly enforced by database

```text
PRIMARY KEY
FOREIGN KEY
UNIQUE
NOT NULL
CHECK
EXCLUDE
PARTIAL UNIQUE INDEX
```

digunakan untuk:

```text
identity
state values
amount positivity
whole-unit money
date ordering
range overlap
account uniqueness
one completed settlement
one OPEN academic year
```

Status:

```text
GOOD
```

## Service/domain enforced

```text
role-dependent behavior
student lifecycle
settlement workflow
transaction balance
transaction authorization
class transfer approval
academic year transition
password/security rules
session semantics
```

Status:

```text
EXPECTED
```

## Cross-table invariants not database-enforced

```text
academic year overlap
history period within academic year
transaction date within academic year
teacher assignment period within academic year
```

Status:

```text
NEEDS CONTRACT DECISION / OWNERSHIP
```

---

# 28. FINAL ASSESSMENT

Database schema secara umum sudah kuat.

Tidak ditemukan:

```text
CONFLICT
```

antara migration final dan domain state machine utama.

Sebagian besar business invariant penting sudah memiliki database enforcement.

Temuan yang tersisa terutama merupakan cross-table temporal rules:

```text
AcademicYear
      ↓
StudentClassHistory
      ↓
Transaction
```

serta beberapa state-dependent constraints:

```text
ClassTransferRequest
SavingsSettlement lineage
```

Temuan tersebut tidak boleh diselesaikan dengan menambahkan constraint secara spontan.

---

# 29. FASE 16.2 STATUS

```text
FASE 16.2 DATABASE CONTRACT AUDIT

Status: COMPLETE

Migration audit:
    COMPLETE

Schema audit:
    COMPLETE

Constraint audit:
    COMPLETE

Foreign key audit:
    COMPLETE

Index audit:
    COMPLETE

Migration up/down audit:
    COMPLETE

Source modification:
    NONE

Migration modification:
    NONE

Fatal conflicts:
    0

Contract decisions required:
    3

Implementation gaps:
    2

Deferred:
    1
```

---

# 30. Definition of Done

- [x] Semua migration diaudit.
- [x] Semua table utama diaudit.
- [x] PRIMARY KEY diaudit.
- [x] FOREIGN KEY diaudit.
- [x] NOT NULL diaudit.
- [x] CHECK constraint diaudit.
- [x] UNIQUE constraint/index diaudit.
- [x] EXCLUDE constraint diaudit.
- [x] Partial unique index diaudit.
- [x] Index access pattern diaudit.
- [x] State consistency diaudit.
- [x] Money precision contract diaudit.
- [x] Date/range contract diaudit.
- [x] Session schema diaudit.
- [x] Migration UP/DOWN diaudit.
- [x] Contract gaps dicatat.
- [x] Tidak ada migration yang diubah.
- [x] Tidak ada source code yang diubah.
- [x] Tidak ada fatal conflict yang ditemukan.

---

# 31. NEXT

```text
FASE 16.3 — Lifecycle Contract Audit
```

Fokus berikutnya:

```text
Student lifecycle
        ↓
Class placement
        ↓
Class transfer
        ↓
Savings account lifecycle
        ↓
Transaction lifecycle
        ↓
Settlement lifecycle
        ↓
Academic year lifecycle
```

FASE 16.3 akan memeriksa apakah seluruh state transition dan urutan lifecycle antar aggregate konsisten.

