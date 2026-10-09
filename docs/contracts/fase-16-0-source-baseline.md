# FASE 16.0 — Source Baseline

## Project

**Tabungan Siswa**

## Baseline Source

```text
tabungan-siswa.tar(20261008-034155).gz
```

## Baseline Commit

```text
1a680c8 feat(ui): complete web foundation and form patterns
```

## Tujuan

FASE 16.0 menetapkan kondisi source aktual sebelum pekerjaan
Re-baseline & Contract Audit dilanjutkan.

Dokumen ini bukan business contract.

Dokumen ini adalah snapshot kondisi awal yang menjadi referensi
untuk subfase FASE 16 berikutnya.

---

# 1. Repository Baseline

Kondisi repository pada snapshot:

```text
working tree: clean
HEAD: 1a680c8
branch: main
```

Commit terakhir:

```text
1a680c8 feat(ui): complete web foundation and form patterns
```

UI foundation telah masuk repository dan tidak perlu dibangun ulang.

---

# 2. Technology Baseline

Stack aktual:

```text
Go
net/http
html/template
HTMX
Tailwind CSS
PostgreSQL
database/sql
pgx
Git/GitHub
```

Rendering:

```text
html/template
```

Bukan:

```text
templ
```

HTMX digunakan sebagai progressive enhancement.

Tailwind digunakan sebagai styling layer.

TailAdmin digunakan sebagai sumber visual/pattern,
bukan sebagai application framework.

---

# 3. Project Structure Baseline

Area utama source:

```text
cmd/
internal/
    app/
    domain/
    http/
    repository/
    security/
    service/
migrations/
assets/
static/
docs/
web/
```

Domain:

```text
internal/domain/
```

Repository:

```text
internal/repository/
internal/repository/postgres/
```

Service:

```text
internal/service/
```

Security:

```text
internal/security/
```

HTTP:

```text
internal/http/
```

Template:

```text
internal/app/templates/
```

Assets:

```text
assets/
static/
```

---

# 4. Domain Baseline

Domain entity/value object yang tersedia:

```text
User
Session
AcademicYear
Class
Student
StudentClassHistory
TeacherClassAssignment
ClassTransferRequest
SavingsAccount
Transaction
SavingsSettlement
AuditLog
Balance
Money
DateRange
```

Domain test tersedia untuk sebagian besar entity/value object.

---

# 5. Repository Baseline

Repository contract yang tersedia mencakup:

```text
User
Session
AcademicYear
Class
Student
StudentClassHistory
TeacherClassAssignment
ClassTransferRequest
SavingsAccount
Transaction
Settlement
Audit
UnitOfWork
```

Implementasi PostgreSQL tersedia di:

```text
internal/repository/postgres/
```

Repository test tersedia untuk area-area tersebut.

---

# 6. Service Baseline

Service yang tersedia:

```text
Authentication
User
AcademicYear
Class
ClassAssignment
ClassTransfer
Student
SavingsAccount
Transaction
Settlement
Audit
```

Service foundation juga menyediakan:

```text
Actor
AuthContext
Authorization
Scope
TransactionScope
UnitOfWork integration
```

---

# 7. Security Baseline

Security components yang tersedia:

```text
Argon2id
Password handling
Session token
Session cookie
Session expiration
CSRF token
Authentication middleware
Authorization helpers
```

Authentication flow HTTP sudah tersedia untuk:

```text
GET  /login
POST /login
POST /logout
GET  /protected
```

---

# 8. HTTP Baseline

HTTP foundation yang tersedia:

```text
authentication handler
login page handler
dashboard page handler
router
middleware
static asset serving
server lifecycle
graceful shutdown
HTTP error handling
```

Feature HTTP untuk domain utama belum lengkap.

Belum tersedia sebagai vertical slice:

```text
Academic Management
Student Management
Savings Account
Passbook
Transaction
Transaction Correction
Settlement
Reporting
User Administration
```

---

# 9. Presentation Baseline

Presentation foundation yang sudah tersedia:

```text
html/template rendering
base layout
header
sidebar
navigation
responsive shell
login page
dashboard shell
HTMX foundation
Tailwind foundation
static CSS
form UI patterns
```

Form pattern yang sudah tersedia:

```text
form-label
form-input
form-input-error
form-select
form-textarea
form-help
form-error
form-actions
form-button-primary
form-button-secondary
```

Dokumentasi form berada di:

```text
docs/ui/form-patterns.md
```

---

# 10. Migration Baseline

Migration yang tersedia:

```text
000001_init_schema
000002_align_domain_contract
000003_require_student_identity
000004_sessions
```

Migration terbaru:

```text
000004_sessions
```

Migration student identity menetapkan:

```text
students.nis  NOT NULL
students.nisn NOT NULL
```

Migration sessions menyediakan persistence untuk session.

---

# 11. Current Business/Domain Areas

Source sudah memiliki implementation untuk:

```text
User
Academic Year
Class
Teacher Assignment
Student
Student Class History
Class Transfer
Savings Account
Transaction
Settlement
Audit
Authentication
Authorization
```

Namun keberadaan implementation tidak otomatis berarti seluruh contract
sudah final.

FASE 16 harus melakukan audit silang:

```text
Domain
    ↕
Migration
    ↕
Repository
    ↕
Service
    ↕
Tests
```

---

# 12. Current Known Gaps

Temuan pada baseline tidak langsung diperbaiki dalam FASE 16.0.

Temuan dicatat untuk dianalisis pada subfase FASE 16 berikutnya.

## GAP-01 — Initial Student Class Assignment

Source memiliki:

```text
StudentClassHistory
ClassTransferRequest
ClassTransferService
```

tetapi baseline belum memiliki use case service yang eksplisit
untuk initial assignment student ke class.

Status:

```text
IMPLEMENTATION GAP
```

Pemilik:

```text
FASE 19 — Student Management
```

---

## GAP-02 — Transaction Date Boundary

Transaction service sudah melakukan:

```text
account authorization
account locking
balance validation
transaction mutation
audit
```

Namun contract transaction date terhadap academic year
harus dipastikan secara eksplisit.

Target contract:

```text
academic_year.start_date
<=
transaction_date
<
academic_year.end_date
```

Status:

```text
CONTRACT TO FREEZE
```

Pemilik implementation:

```text
FASE 21 — Transaction Engine
```

---

## GAP-03 — Audit Coverage

Audit sudah digunakan pada beberapa mutation penting.

Namun coverage seluruh business mutation harus diverifikasi.

Area yang harus diaudit:

```text
User
AcademicYear
Class
Student
StudentClassHistory
TeacherClassAssignment
ClassTransferRequest
SavingsAccount
Transaction
Settlement
```

Status:

```text
AUDIT COVERAGE GAP
```

Pemilik:

```text
FASE 22 — Transaction Correction & Audit
```

---

## GAP-04 — Academic Year Period Overlap

Database sudah memiliki constraint:

```text
start_date < end_date
```

dan invariant untuk satu academic year OPEN.

Namun baseline harus memastikan apakah periode academic year
yang overlap diperbolehkan atau tidak.

Status:

```text
CONTRACT DECISION REQUIRED
```

Jangan mengubah migration sebelum contract diputuskan.

---

## GAP-05 — Settlement Reopen Workflow

Baseline memiliki:

```text
ADMIN → Reopen Settlement
```

Belum ada workflow:

```text
WALI_KELAS
    ↓
Request Reopen
    ↓
ADMIN
    ↓
Approve
```

Status:

```text
NOT PART OF CURRENT CONTRACT
```

Jangan membuat UI untuk workflow tersebut sebelum contract baru dibuat.

---

# 13. Current Test Baseline

Test source tersedia pada:

```text
internal/domain/
internal/repository/
internal/repository/postgres/
internal/service/
internal/security/
internal/http/
```

Baseline verification yang wajib dijalankan dari development environment:

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/web
git diff --check
```

Catatan:

FASE 16.0 tidak mengubah source sehingga test baseline digunakan
untuk memastikan snapshot source tidak berubah selama audit.

---

# 14. Current Architecture Baseline

Architecture:

```text
HTTP
 ↓
Service
 ↓
Repository
 ↓
PostgreSQL
```

Composition root:

```text
internal/app/
```

Domain tidak bergantung pada:

```text
HTTP
PostgreSQL
template
HTMX
Tailwind
```

---

# 15. FASE 16.0 Boundary

FASE 16.0 BOLEH:

```text
inspect source
inspect migrations
inspect tests
inventory architecture
record baseline
record gaps
```

FASE 16.0 TIDAK BOLEH:

```text
menambahkan HTTP feature
membuat Student UI
membuat Transaction UI
membuat Settlement UI
menambahkan HTMX feature
mengubah business rule tanpa contract review
```

Contract-level migration fix baru boleh dilakukan setelah
subfase contract menentukan invariant tersebut.

---

# 16.0 Definition of Done

FASE 16.0 selesai jika:

```text
[✓] Source snapshot ditetapkan
[✓] Baseline commit ditetapkan
[✓] Technology baseline ditetapkan
[✓] Architecture baseline ditetapkan
[✓] Domain inventory ditetapkan
[✓] Repository inventory ditetapkan
[✓] Service inventory ditetapkan
[✓] Security inventory ditetapkan
[✓] HTTP inventory ditetapkan
[✓] Presentation inventory ditetapkan
[✓] Migration inventory ditetapkan
[✓] Known gaps dicatat
[✓] Setiap gap memiliki kategori
[✓] Setiap implementation gap memiliki fase pemilik
```

Verification environment:

```text
[ ] go test ./...
[ ] go test -race ./...
[ ] go vet ./...
[ ] go build ./cmd/web
[ ] git diff --check
```

---

# 17. Next Step

Setelah FASE 16.0:

```text
FASE 16.1 — Domain Contract Audit
```

Fokus:

```text
User
AcademicYear
Class
Student
StudentClassHistory
TeacherClassAssignment
ClassTransferRequest
SavingsAccount
Transaction
SavingsSettlement
AuditLog
Session
```

Untuk setiap domain entity:

```text
fields
invariants
relations
lifecycle
mutation
deletion
authorization
```

Tidak masuk HTTP.

Tidak masuk UI.

Tidak masuk HTMX.

---

# 18. Baseline Principle

FASE 16.0 adalah titik pembanding.

Jika setelah seluruh FASE 16 selesai terjadi perubahan terhadap
kontrak, perubahan tersebut harus dapat dijelaskan:

```text
BASELINE
    ↓
CONTRACT DECISION
    ↓
IMPLEMENTATION
    ↓
TEST
```

Bukan:

```text
kode berubah
    ↓
baru mencari tahu kontraknya
```

---

# STATUS

```text
FASE 16.0 — SOURCE BASELINE

SOURCE INVENTORY       COMPLETE
ARCHITECTURE INVENTORY COMPLETE
KNOWN GAP REGISTER     COMPLETE
CONTRACT AUDIT         NEXT
IMPLEMENTATION         NONE
UI CHANGE              NONE
```

Next:

```text
FASE 16.1 — Domain Contract Audit
```
