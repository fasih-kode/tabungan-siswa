# FASE 16.4 — AUTHORIZATION CONTRACT AUDIT

Status: AUDIT COMPLETE
Phase: 16.4
Parent: FASE 16 — CONTRACT HARDENING
Scope: Roles, permissions, scope, ownership, resource authorization and mutation authorization
Source baseline: commit `1a680c8`
Audit basis: source snapshot `tabungan-siswa.tar(20261008-034155).gz`

---

# 1. Tujuan

FASE 16.4 melakukan audit terhadap authorization contract aplikasi Tabungan Siswa.

Audit memeriksa hubungan:

```text
ACTOR
  ↓
ROLE
  ↓
PERMISSION
  ↓
RESOURCE
  ↓
SCOPE
  ↓
MUTATION
```

Authorization tidak hanya dinilai dari role.

Untuk resource yang memiliki scope, authorization harus memastikan:

```text
Actor
+
Target resource
+
Academic Year
+
Class / Student ownership
```

tetap konsisten.

Audit meliputi:

1. Role validation.
2. Self ownership.
3. Class scope.
4. Student scope.
5. Resource ownership.
6. Academic-year scope.
7. Read authorization.
8. Mutation authorization.
9. Cross-resource authorization.
10. Role escalation.
11. IDOR-style access melalui identifier.
12. Scope propagation antar service.
13. Authorization tests.

FASE 16.4 tidak mengubah source code.

---

# 2. Authorization Contract

Role yang dikunci:

```text
ADMIN
WALI_KELAS
SISWA
```

## ADMIN

Secara umum:

```text
global application scope
```

Admin tidak dibatasi class/student scope untuk resource administratif.

## WALI_KELAS

Scope:

```text
class assignment
+
academic year
```

Wali kelas hanya dapat mengakses student melalui class yang menjadi scope-nya pada academic year terkait.

## SISWA

Scope:

```text
self
```

Student hanya boleh mengakses resource yang berhubungan dengan dirinya sendiri.

---

# 3. Authorization Primitives

Implementasi memiliki tiga primitive utama:

```text
RequireRole()
RequireSelf()
RequireClassScope()
RequireStudentScope()
```

---

# 4. RequireRole

Implementasi:

```text
actor.Validate()
      ↓
actor.Role ∈ allowed roles
```

Jika tidak cocok:

```text
ErrForbidden
```

## 4.1 Contract

`RequireRole` tidak melakukan authorization terhadap resource.

Fungsinya hanya:

```text
role gate
```

Ini benar.

Resource scope tetap harus dilakukan oleh service.

Status:

```text
MATCH
```

---

# 5. RequireSelf

Contract:

```text
actor.UserID == targetUserID
```

Syarat:

```text
Actor valid
Target ID != nil
```

Mismatch:

```text
ErrScopeViolation
```

Status:

```text
MATCH
```

---

# 6. RequireClassScope

Contract:

```text
ADMIN
    → global access

WALI_KELAS
    → class must be assigned to actor
       in requested academic year

SISWA
    → forbidden
```

Implementasi:

```text
ListByClassAndAcademicYear
        ↓
assignment.UserID == actor.UserID
```

Status:

```text
MATCH
```

## 6.1 Important observation

Scope tidak hanya berdasarkan:

```text
classID
```

tetapi:

```text
classID + academicYearID
```

Ini benar dan penting.

Wali kelas yang ditugaskan pada:

```text
Class A / AY 2026
```

tidak otomatis mendapatkan scope:

```text
Class A / AY 2027
```

Status:

```text
MATCH
```

---

# 7. RequireStudentScope

Contract:

```text
ADMIN
    → global access

WALI_KELAS
    → student must have a class history
       within requested academic year
       whose class is assigned to actor

SISWA
    → forbidden
```

Implementasi:

```text
StudentClassHistories.ListByStudentAndAcademicYear()
        ↓
for each history
        ↓
RequireClassScope()
        ↓
class assigned to actor?
```

Status:

```text
MATCH
```

## 7.1 Scope propagation

Student scope tidak diberikan langsung berdasarkan:

```text
studentID
```

tetapi diturunkan melalui:

```text
student
  ↓
class history
  ↓
class
  ↓
teacher assignment
  ↓
actor
```

Ini merupakan authorization chain yang benar.

---

# 8. Scope Isolation

Wali kelas dengan:

```text
Class A / AY 2026
```

tidak boleh mendapatkan access hanya karena:

```text
StudentID valid
```

Access harus memiliki:

```text
AcademicYearID
```

dan:

```text
Class assignment
```

Status:

```text
MATCH
```

---

# 9. ADMIN AUTHORIZATION MATRIX

| Resource / Action | ADMIN |
|---|---|
| User Create | ALLOW |
| User Update | ALLOW |
| User Get | ALLOW |
| User GetByUsername | ALLOW |
| AcademicYear Create | ALLOW |
| AcademicYear Get | ALLOW |
| AcademicYear List | ALLOW |
| AcademicYear StartClosing | ALLOW |
| AcademicYear Close | ALLOW |
| AcademicYear Reopen | ALLOW |
| Class Create | ALLOW |
| Class Get | ALLOW |
| Class List | ALLOW |
| Class Update | ALLOW |
| Student Create | ALLOW |
| Student Get | ALLOW |
| Student List | ALLOW |
| Student Update | ALLOW |
| Student Leave | ALLOW |
| Teacher Assignment Create | ALLOW |
| Transfer Request | ALLOW |
| Transfer List | ALLOW |
| Transfer Approve | ALLOW |
| Transfer Reject | ALLOW |
| Savings Account Create | ALLOW |
| Savings Account Get | ALLOW |
| Savings Account Balance | ALLOW |
| Transaction Create | ALLOW |
| Transaction Get | ALLOW |
| Transaction List | ALLOW |
| Transaction Edit | ALLOW |
| Transaction Cancel | ALLOW |
| Settlement Create | ALLOW |
| Settlement Get | ALLOW |
| Settlement List | ALLOW |
| Settlement Reopen | ALLOW |
| Audit Read | ALLOW |

Status:

```text
MATCH
```

---

# 10. WALI_KELAS AUTHORIZATION MATRIX

| Resource / Action | WALI_KELAS |
|---|---|
| User Get | Scoped |
| User GetByUsername | DENY |
| User Create | DENY |
| User Update | DENY |
| AcademicYear Get | ALLOW |
| AcademicYear List | ALLOW |
| AcademicYear Create | DENY |
| AcademicYear Close | DENY |
| Class List | Own assigned classes |
| Class Get | DENY |
| Class Create | DENY |
| Class Update | DENY |
| Student Get | Student scope |
| Student List | Class scope |
| Student Create | DENY |
| Student Update | DENY |
| Student Leave | DENY |
| Teacher Assignment Read | Class scope |
| Teacher Assignment Create | DENY |
| Transfer Request | Student scope |
| Transfer Get | Student scope |
| Transfer List | Student scope |
| Transfer Pending List | DENY |
| Transfer Approve | DENY |
| Transfer Reject | DENY |
| Savings Account Create | Student scope |
| Savings Account Get | Student scope |
| Savings Account Balance | Student scope |
| Transaction Create | Student/account scope |
| Transaction Get | Student/account scope |
| Transaction List | Student/account scope |
| Transaction Edit | Student/account scope |
| Transaction Cancel | Student/account scope |
| Settlement Create | Student/account scope |
| Settlement Get | Student/account scope |
| Settlement List | Student/account scope |
| Settlement Reopen | DENY |
| Audit Read | DENY |

Status:

```text
MATCH
```

---

# 11. SISWA AUTHORIZATION MATRIX

| Resource / Action | SISWA |
|---|---|
| User Get | Self |
| User GetByUsername | DENY |
| User Create | DENY |
| User Update | DENY |
| AcademicYear Get | ALLOW |
| AcademicYear List | ALLOW |
| AcademicYear Create | DENY |
| AcademicYear Close | DENY |
| AcademicYear Reopen | DENY |
| Class List | Own class |
| Class Get | DENY |
| Class Create | DENY |
| Class Update | DENY |
| Student Get | Self |
| Student GetByNIS | Self |
| Student GetByNISN | Self |
| Student GetByUserID | Self |
| Student List | DENY |
| Student Create | DENY |
| Student Update | DENY |
| Student Leave | DENY |
| Teacher Assignment Read | Own class scope |
| Teacher Assignment Create | DENY |
| Transfer Request | DENY |
| Transfer Get | Self |
| Transfer List | Self |
| Transfer Pending List | DENY |
| Transfer Approve | DENY |
| Transfer Reject | DENY |
| Savings Account Create | DENY |
| Savings Account Get | Self |
| Savings Account Balance | Self |
| Transaction Create | DENY |
| Transaction Get | Self |
| Transaction List | Self |
| Transaction Edit | DENY |
| Transaction Cancel | DENY |
| Settlement Create | DENY |
| Settlement Get | Self |
| Settlement List | Self |
| Settlement Reopen | DENY |
| Audit Read | DENY |

Status:

```text
MATCH
```

---

# 12. USER SERVICE

## Get

Authorization:

```text
ADMIN
    → any user

SISWA
    → RequireSelf()

WALI_KELAS
    → resolve target user
    → resolve student
    → RequireStudentScope()
```

Status:

```text
MATCH
```

## GetByUsername

```text
ADMIN only
```

Status:

```text
MATCH
```

## Create

```text
ADMIN only
```

Status:

```text
MATCH
```

## Update

```text
ADMIN only
```

Status:

```text
MATCH
```

No self-service mutation exists.

Status:

```text
MATCH
```

---

# 13. ACADEMIC YEAR SERVICE

## Read

```text
ADMIN
WALI_KELAS
SISWA
```

allowed.

## Mutation

```text
Create
StartClosing
Close
Reopen
```

ADMIN only.

Status:

```text
MATCH
```

## 13.1 Scope observation

Academic year is currently treated as globally readable metadata.

No role-specific academic-year ownership is required by current contract.

Status:

```text
MATCH
```

---

# 14. CLASS SERVICE

## Create / Get / Update

```text
ADMIN
```

only.

## List

### ADMIN

All classes.

### WALI_KELAS

Only classes returned by:

```text
ListByTeacherAndAcademicYear(
    actor.UserID,
    academicYearID
)
```

### SISWA

Only classes represented in:

```text
StudentClassHistory
```

for:

```text
actor's student
+
requested academic year
```

Status:

```text
MATCH
```

---

# 15. STUDENT SERVICE

## Get

### ADMIN

Global.

### SISWA

Requires:

```text
student.UserID == actor.UserID
```

### WALI_KELAS

Requires:

```text
RequireStudentScope()
```

Status:

```text
MATCH
```

---

# 16. STUDENT LOOKUP BY NIS

GetByNIS follows the same ownership policy.

### ADMIN

Global.

### SISWA

Student returned by NIS must belong to actor.

### WALI_KELAS

Student returned by NIS must belong to actor scope.

Status:

```text
MATCH
```

---

# 17. STUDENT LOOKUP BY NISN

Same contract:

```text
ADMIN → global
WALI_KELAS → student scope
SISWA → self
```

Status:

```text
MATCH
```

---

# 18. STUDENT LOOKUP BY USER ID

### ADMIN

Any user ID.

### SISWA

```text
RequireSelf(actor, userID)
```

### WALI_KELAS

```text
userID
  ↓
Student
  ↓
RequireStudentScope
```

Status:

```text
MATCH
```

---

# 19. STUDENT LIST

Required input:

```text
AcademicYearID
ClassID
```

### ADMIN

Allowed.

### WALI_KELAS

Before repository query:

```text
RequireClassScope()
```

### SISWA

Rejected.

Status:

```text
MATCH
```

This is an important protection because the repository itself is not relied upon to determine authorization.

---

# 20. STUDENT MUTATIONS

Create:

```text
ADMIN only
```

Update:

```text
ADMIN only
```

Leave:

```text
ADMIN only
```

WALI_KELAS cannot mutate student master data.

SISWA cannot mutate student master data.

Status:

```text
MATCH
```

---

# 21. TEACHER CLASS ASSIGNMENT

## Assign

```text
ADMIN only
```

Additionally:

```text
target user must have RoleWaliKelas
```

This prevents assigning a student/admin as teacher assignment.

Status:

```text
MATCH
```

---

# 22. TEACHER ASSIGNMENT GET

Authorization is resource-based:

```text
ADMIN
    → any assignment

WALI_KELAS
    → RequireClassScope()

SISWA
    → require student class scope
```

Status:

```text
MATCH
```

---

# 23. TEACHER ASSIGNMENT LIST BY TEACHER

Implementation first loads:

```text
target teacher + academic year
```

Then:

### ADMIN

Returns all target teacher assignments.

### WALI_KELAS

Filters results through:

```text
RequireClassScope()
```

### SISWA

Filters results through:

```text
student class scope
```

Status:

```text
MATCH
```

## 23.1 Contract observation

A WALI_KELAS may provide another teacher's UserID and receive assignments for classes that are within the caller's own class scope.

This is not privilege escalation if the contract allows a wali to see teacher assignments for their own class.

Current source does not define a stricter:

```text
self-only teacher assignment read
```

rule.

Status:

```text
MATCH
```

under current contract.

---

# 24. TEACHER ASSIGNMENT LIST BY CLASS

### ADMIN

Global.

### WALI_KELAS

Requires:

```text
RequireClassScope()
```

### SISWA

Requires:

```text
student
+
class history
+
academic year
```

Status:

```text
MATCH
```

---

# 25. SAVINGS ACCOUNT AUTHORIZATION

Savings account is a critical cross-resource boundary.

Authorization follows:

```text
Account
  ↓
Student
  ↓
AcademicYear
```

## ADMIN

Global.

## WALI_KELAS

```text
RequireStudentScope(
    studentID,
    academicYearID
)
```

## SISWA

```text
Account.StudentID
    ↓
Student.UserID
    ↓
RequireSelf()
```

Status:

```text
MATCH
```

---

# 26. SAVINGS ACCOUNT CREATE

Allowed:

```text
ADMIN
WALI_KELAS
```

Wali requires:

```text
RequireStudentScope()
```

Student:

```text
DENY
```

Status:

```text
MATCH
```

---

# 27. SAVINGS ACCOUNT READ

Get by account ID:

```text
ADMIN
    → global

WALI_KELAS
    → student scope

SISWA
    → student self
```

Get by:

```text
StudentID + AcademicYearID
```

uses the same policy.

Status:

```text
MATCH
```

This prevents a student from changing only:

```text
AcademicYearID
```

and using another student's account.

---

# 28. SAVINGS ACCOUNT BALANCE

Same resource authorization as account read:

```text
ADMIN
WALI_KELAS scoped
SISWA self
```

Status:

```text
MATCH
```

---

# 29. TRANSACTION AUTHORIZATION

Transaction does not authorize directly from:

```text
Transaction.CreatedBy
```

Instead:

```text
Transaction
    ↓ SavingsAccount
    ↓ Student
    ↓ AcademicYear
```

Then:

```text
authorizeAccount()
```

is applied.

This is correct because the resource owner is the savings account/student, not the user who originally entered the transaction.

Status:

```text
MATCH
```

---

# 30. TRANSACTION READ

## Get

```text
transaction ID
    ↓
account lookup
    ↓
authorizeAccount
```

## List

```text
account ID
    ↓
account lookup
    ↓
authorizeAccount
```

Status:

```text
MATCH
```

This prevents direct transaction-ID or account-ID access from bypassing student/class scope.

---

# 31. TRANSACTION CREATE

Allowed:

```text
ADMIN
WALI_KELAS
```

Not allowed:

```text
SISWA
```

Wali authorization:

```text
account
  ↓
student
  ↓
academic year
  ↓
RequireStudentScope
```

Status:

```text
MATCH
```

---

# 32. TRANSACTION EDIT

Allowed:

```text
ADMIN
WALI_KELAS
```

Wali must pass account scope.

Student cannot edit.

Status:

```text
MATCH
```

---

# 33. TRANSACTION CANCEL

Allowed:

```text
ADMIN
WALI_KELAS
```

Wali must pass account scope.

Student cannot cancel.

Status:

```text
MATCH
```

---

# 34. SETTLEMENT AUTHORIZATION

Settlement uses the same account ownership chain.

```text
Settlement
    ↓
SavingsAccount
    ↓
Student
    ↓
AcademicYear
```

## Read

```text
ADMIN
WALI_KELAS scoped
SISWA self
```

## Create

```text
ADMIN
WALI_KELAS scoped
```

## Reopen

```text
ADMIN only
```

Status:

```text
MATCH
```

---

# 35. YEAR-END SETTLEMENT

Wali may settle an account only if:

```text
account
+
student
+
academic year
```

falls inside their student scope.

Additionally:

```text
AcademicYear = CLOSING
```

is required.

Status:

```text
MATCH
```

---

# 36. STUDENT-LEAVING SETTLEMENT

Wali may settle only within:

```text
student scope
```

and:

```text
Student = LEFT
```

is required.

Status:

```text
MATCH
```

---

# 37. SETTLEMENT REOPEN

```text
ADMIN only
```

No WALI_KELAS path exists.

No SISWA path exists.

Status:

```text
MATCH
```

## 37.1 Business workflow observation

There is no:

```text
WALI_KELAS request
      ↓
ADMIN approval
```

workflow.

This is not currently an authorization bug.

It is a business-process decision already identified as:

```text
GAP-SETTLEMENT-REOPEN-01
```

Status:

```text
DEFERRED
```

---

# 38. CLASS TRANSFER AUTHORIZATION

## Request

### ADMIN

Allowed.

### WALI_KELAS

Allowed only when:

```text
student
+
academic year
```

is in caller's scope.

### SISWA

Explicitly rejected.

Status:

```text
MATCH
```

---

# 39. CLASS TRANSFER READ

Get:

```text
ADMIN
WALI_KELAS scoped
SISWA self
```

List by student:

```text
ADMIN
WALI_KELAS scoped
SISWA self
```

Status:

```text
MATCH
```

---

# 40. CLASS TRANSFER APPROVE / REJECT

Both:

```text
ADMIN only
```

Wali cannot directly approve or reject.

Student cannot.

Status:

```text
MATCH
```

---

# 41. PENDING TRANSFER LIST

```text
ADMIN only
```

This is correct because pending transfer review is an administrative workflow in the current contract.

Status:

```text
MATCH
```

---

# 42. AUDIT AUTHORIZATION

Audit read operations:

```text
ListByEntity
ListByActor
```

are:

```text
ADMIN only
```

WALI_KELAS:

```text
DENY
```

SISWA:

```text
DENY
```

Status:

```text
MATCH
```

This matches the locked audit contract.

---

# 43. AUDIT WRITE PATH

Internal `recordAudit()` validates:

```text
Actor
EntityID
```

but does not independently enforce:

```text
ADMIN only
```

This is intentional.

Mutation services call it using the already-authorized actor and their own mutation authorization.

Audit recording is therefore:

```text
internal mutation concern
```

rather than:

```text
public authorization endpoint
```

Status:

```text
MATCH
```

---

# 44. IDOR AUDIT

Potential IDOR vectors audited:

```text
userID
studentID
NIS
NISN
classID
academicYearID
teacherAssignmentID
transferRequestID
savingsAccountID
transactionID
settlementID
```

## Result

Resource reads do not rely solely on identifier validity.

They resolve the target resource and then perform authorization.

Examples:

```text
transactionID
    ↓
account
    ↓
student
    ↓
scope
```

and:

```text
settlementID
    ↓
account
    ↓
student
    ↓
scope
```

Status:

```text
MATCH
```

---

# 45. ACADEMIC YEAR SCOPE

Academic year is explicitly included in:

```text
RequireClassScope()
RequireStudentScope()
```

This prevents:

```text
WALI_KELAS assignment AY-A
```

from automatically granting:

```text
same class AY-B
```

access.

Status:

```text
MATCH
```

---

# 46. CLASS SCOPE PROPAGATION

Authorization chain:

```text
WALI_KELAS
     ↓
TeacherClassAssignment
     ↓
Class
     ↓
AcademicYear
```

Student scope:

```text
WALI_KELAS
     ↓
StudentClassHistory
     ↓
Class
     ↓
AcademicYear
     ↓
TeacherClassAssignment
     ↓
WALI_KELAS
```

Status:

```text
MATCH
```

---

# 47. HISTORICAL CLASS SCOPE

`RequireStudentScope()` checks all student class histories for the requested academic year.

Therefore a WALI_KELAS who is assigned to a class that the student occupied earlier in the same academic year can still pass student scope.

Example:

```text
AY 2026

Student A:
    Class 7A
        ↓ transfer
    Class 7B

Wali X:
    assigned 7A

Wali Y:
    assigned 7B
```

Both may satisfy `RequireStudentScope()` for Student A in AY 2026 because both classes exist in the student's history.

## Contract status

This is not automatically a security flaw.

The current scope contract is:

```text
student belongs to class history
within academic year
```

not explicitly:

```text
student's current class only
```

Status:

```text
GAP - CONTRACT DECISION
```

ID:

```text
GAP-AUTH-HISTORY-01
```

Decision required:

```text
Apakah wali kelas hanya boleh mengakses
current class student, atau seluruh class history
dalam academic year?
```

Owner:

```text
FASE 16.7
```

---

# 48. STUDENT SELF SCOPE

Siswa menggunakan:

```text
Student.UserID
```

sebagai ownership anchor.

Ini lebih aman daripada menggunakan:

```text
StudentID == Actor.UserID
```

karena kedua identifier berbeda domain.

Status:

```text
MATCH
```

---

# 49. STUDENT WITHOUT USER ACCOUNT

Jika:

```text
student.UserID == nil
```

Siswa tidak dapat memperoleh ownership terhadap student tersebut.

Service mengembalikan:

```text
ErrScopeViolation
```

Status:

```text
MATCH
```

---

# 50. INVALID ACTOR

`Actor.Validate()` menjadi prerequisite pada:

```text
RequireRole
RequireSelf
RequireClassScope
RequireStudentScope
```

Invalid actor:

```text
UserID = nil
```

ditolak.

Status:

```text
MATCH
```

---

# 51. ROLE ESCALATION

Actor role berasal dari:

```text
service.Actor
```

dan authorization dilakukan berdasarkan role tersebut.

Tidak ditemukan jalur dalam service yang:

```text
WALI_KELAS → ADMIN
SISWA → WALI_KELAS
SISWA → ADMIN
```

secara otomatis.

Status:

```text
MATCH
```

Authentication/security layer bertanggung jawab memastikan actor yang masuk ke service berasal dari authenticated identity.

---

# 52. AUTHORIZATION TEST COVERAGE

Test yang tersedia mencakup:

```text
RequireRole allowed
RequireRole forbidden
RequireRole invalid actor

RequireSelf own
RequireSelf other
RequireSelf nil target
RequireSelf invalid actor

RequireClassScope invalid actor
RequireClassScope nil IDs
RequireClassScope SISWA forbidden

RequireStudentScope invalid actor
RequireStudentScope nil IDs
RequireStudentScope SISWA forbidden
```

Service tests juga mencakup:

```text
Student self
Student other rejected
Wali scope
Transaction Wali in scope
Transaction Wali out of scope
Settlement Wali in scope
Settlement Student rejected
Class assignment Wali scope
Transfer Student rejected
Transfer non-admin approval rejected
Audit non-admin access
```

Status:

```text
MATCH
```

---

# 53. AUTHORIZATION MATRIX SUMMARY

```text
                         ADMIN   WALI_KELAS   SISWA
-----------------------------------------------------
Global administration      ✓         ✗          ✗
Global finance             ✓         ✗          ✗
Student self               ✓         ✓*         ✓
Student list               ✓         ✓*         ✗
Class list                 ✓         ✓*         ✓*
Transaction read           ✓         ✓*         ✓*
Transaction mutation       ✓         ✓*         ✗
Settlement read            ✓         ✓*         ✓*
Settlement mutation        ✓         ✓*         ✗
Settlement reopen          ✓         ✗          ✗
Transfer read              ✓         ✓*         ✓*
Transfer request           ✓         ✓*         ✗
Transfer approve/reject    ✓         ✗          ✗
Audit read                 ✓         ✗          ✗
```

`*` = scope restricted.

---

# 54. FINDINGS

## AUTH-01 — Role Gate

```text
RequireRole()
```

is consistent.

Status:

```text
MATCH
```

---

## AUTH-02 — Self Ownership

```text
RequireSelf()
```

is consistent.

Status:

```text
MATCH
```

---

## AUTH-03 — Class Scope

Class scope includes:

```text
classID + academicYearID + teacher assignment
```

Status:

```text
MATCH
```

---

## AUTH-04 — Student Scope

Student scope derives through:

```text
student class history
→ class
→ teacher assignment
```

Status:

```text
MATCH
```

---

## AUTH-05 — Financial Resource Scope

Transaction and settlement authorization derives through:

```text
account
→ student
→ academic year
```

Status:

```text
MATCH
```

---

## AUTH-06 — Audit Access

Audit read is ADMIN only.

Status:

```text
MATCH
```

---

## AUTH-07 — Student Historical Class Scope

Wali can potentially access a student through any class history within the same academic year, not necessarily only the current class.

Status:

```text
GAP - CONTRACT DECISION
```

ID:

```text
GAP-AUTH-HISTORY-01
```

Owner:

```text
FASE 16.7
```

---

# 55. NO FATAL AUTHORIZATION CONFLICT

Tidak ditemukan jalur yang secara jelas memungkinkan:

```text
SISWA
```

mengakses:

```text
student lain
account siswa lain
transaction siswa lain
settlement siswa lain
```

atau:

```text
WALI_KELAS
```

mengakses student di luar class scope.

Tidak ditemukan jalur:

```text
WALI_KELAS → ADMIN mutation
```

pada service yang diaudit.

Status:

```text
FATAL CONFLICTS = 0
```

---

# 56. CONTRACT DECISION REQUIRED

Hanya satu temuan authorization yang membutuhkan keputusan bisnis eksplisit:

```text
GAP-AUTH-HISTORY-01
```

Pertanyaan:

```text
Wali kelas memiliki akses student berdasarkan:

A.
seluruh StudentClassHistory dalam academic year

atau

B.
hanya current StudentClassHistory
```

Implementasi saat ini menggunakan:

```text
A
```

Tidak boleh diubah sebelum kontrak diputuskan.

---

# 57. IMPORTANT NON-FINDINGS

Hal-hal berikut bukan authorization bug:

## 57.1 Wali melihat teacher assignment guru lain

Selama assignment tersebut berada pada class yang menjadi scope Wali, hal ini tidak melampaui class scope.

## 57.2 Student melihat teacher assignment

Student hanya mendapatkan assignment yang berhubungan dengan class history miliknya.

## 57.3 Academic year dapat dibaca semua role

Academic year merupakan metadata global dan current contract tidak membatasi read berdasarkan ownership.

## 57.4 Audit write bukan ADMIN-only

Audit write adalah internal mutation concern.

Audit read tetap ADMIN-only.

---

# 58. AUTHORIZATION CONTRACT CHAIN

Authorization architecture saat ini:

```text
                Actor
                  │
                  ▼
            Actor.Validate
                  │
                  ▼
             Role Check
                  │
        ┌─────────┴─────────┐
        │                   │
      ADMIN              Scoped Role
                            │
                 ┌──────────┴──────────┐
                 │                     │
             WALI_KELAS             SISWA
                 │                     │
                 ▼                     ▼
          Class / Student            Self
              Scope                 Ownership
                 │                     │
                 └──────────┬──────────┘
                            ▼
                         Resource
```

Struktur ini konsisten dengan arsitektur service.

---

# 59. AUTHORIZATION BOUNDARY

Domain tidak mengetahui:

```text
ADMIN
WALI_KELAS
SISWA authorization policy
```

Service mengetahui:

```text
role
scope
resource
```

Repository hanya menyediakan data yang diperlukan untuk mengevaluasi scope.

Status:

```text
MATCH
```

---

# 60. NO DATABASE AUTHORIZATION LEAK

Database tidak menjadi tempat utama authorization.

Service memastikan:

```text
authorized actor
```

sebelum resource diberikan atau dimutasi.

Database tetap memberikan referential integrity.

Status:

```text
MATCH
```

---

# 61. FASE 16.4 FINAL ASSESSMENT

Authorization implementation secara umum sudah konsisten dengan role model:

```text
ADMIN
WALI_KELAS
SISWA
```

Architecture memiliki separation yang jelas:

```text
Role
  ↓
Class Scope
  ↓
Student Scope
  ↓
Resource Scope
```

Financial resources menggunakan:

```text
SavingsAccount
    ↓
Student
    ↓
AcademicYear
```

sebagai authorization chain.

Tidak ditemukan fatal privilege escalation.

Satu contract decision tersisa:

```text
historical StudentClassHistory
vs
current StudentClassHistory
```

untuk menentukan scope WALI_KELAS.

---

# 62. FASE 16.4 STATUS

```text
FASE 16.4 AUTHORIZATION CONTRACT AUDIT

Status: COMPLETE

Role audit:
    COMPLETE

Permission audit:
    COMPLETE

Class scope:
    COMPLETE

Student scope:
    COMPLETE

Resource ownership:
    COMPLETE

Financial resource scope:
    COMPLETE

IDOR audit:
    COMPLETE

Privilege escalation audit:
    COMPLETE

Authorization test coverage:
    COMPLETE

Source modification:
    NONE

Migration modification:
    NONE

Fatal conflicts:
    0

Contract decisions:
    1
```

---

# 63. Definition of Done

- [x] ADMIN authorization diaudit.
- [x] WALI_KELAS authorization diaudit.
- [x] SISWA authorization diaudit.
- [x] RequireRole diaudit.
- [x] RequireSelf diaudit.
- [x] RequireClassScope diaudit.
- [x] RequireStudentScope diaudit.
- [x] User authorization diaudit.
- [x] Academic year authorization diaudit.
- [x] Class authorization diaudit.
- [x] Student authorization diaudit.
- [x] Teacher assignment authorization diaudit.
- [x] Savings account authorization diaudit.
- [x] Transaction authorization diaudit.
- [x] Settlement authorization diaudit.
- [x] Class transfer authorization diaudit.
- [x] Audit authorization diaudit.
- [x] IDOR-style resource access diaudit.
- [x] Privilege escalation path diaudit.
- [x] Scope propagation diaudit.
- [x] Authorization tests reviewed.
- [x] Tidak ada source code yang diubah.
- [x] Tidak ada migration yang diubah.
- [x] Tidak ditemukan fatal authorization conflict.

---

# 64. NEXT

```text
FASE 16.5 — Financial Contract Audit
```

Fokus:

```text
Money
  ↓
Balance
  ↓
Deposit
  ↓
Withdrawal
  ↓
Transaction date
  ↓
Transaction cancellation
  ↓
Settlement
  ↓
Reopen
  ↓
Financial invariants
```

Audit berikutnya akan mengunci aturan finansial sebelum masuk ke implementation phase.

