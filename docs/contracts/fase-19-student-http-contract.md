# FASE 19 — Student Management HTTP Contract

> Keputusan pembuatan kelas/siswa, impor, daftar, dan penempatan pada dokumen ini telah diperbarui oleh [Alur kelas dan penempatan siswa](student-class-placement-workflow.md), 16 Oktober 2026. Jika rincian lama di bawah berbeda, dokumen alur tersebut menjadi acuan.

Status: VERIFIED — READY TO LOCK
Decision approval: product owner, 2026-10-14.

Scope: expose Student management through authenticated server-rendered HTML,
following the locked FASE 17 form and FASE 18 page patterns. The approved
decisions below are implemented and verified against the checks listed below.

## Approved FASE 19 decisions

1. Classes belong to one academic year and have an effective placement date
   within that year's period. Existing student history is not rewritten when
   class configuration changes.
2. Admin creates a student and its initial class history in the same
   transaction. The selected class's effective date starts the history;
   transfers during a year continue through the class-transfer workflow.
3. Leaving is an explicit Admin action with a supplied effective date. Marking
   the student LEFT and closing the relevant active class history must be one
   atomic service operation.
4. Reject leaving while any PENDING class transfer exists for the student, in
   any academic year. Do not cancel or reject transfers implicitly.
5. Leaving does not settle a savings account. The page may guide the Admin to
   the separate settlement flow when needed.
6. Student creation and import do not create or associate login accounts.
   Account creation remains a separate workflow through Pengguna.
7. Class assignment rejects LEFT students. Admin reactivates a LEFT student
   only through an explicit action that requires a year, class, and effective
   date, then creates a new history period atomically with ACTIVE status.
   Existing closed history remains unchanged; overlapping periods are rejected.
8. Admin can list all students, filter by year/class, or filter students
   without a placement for the selected scope. Wali Kelas remains limited to
   an assigned class/year scope; Siswa is redirected to their own detail.
9. Admin may import student master records from `.xlsx`. Validate the complete
   workbook before writing; if any row is invalid, duplicated in the workbook,
   or conflicts with existing NIS/NISN, reject the entire file and report row
   numbers and reasons. Persist each student and its initial placement in one
   transaction only when every row is valid. Import does not create login
   accounts.

## 1. Existing behavior found in the repository

### Student master data

- `StudentService.Create` is Admin-only. It creates an `ACTIVE` student with a
  required nonblank name, NIS, and NISN, then writes the initial class history
  in the same unit of work. It does not create a user or savings account.
- `StudentService.Update` is Admin-only. It updates `UserID`, NIS, NISN, and
  name; it does not change status. NIS, NISN, and name remain required.
- Update validates an optional `user_id` against an existing Siswa account.
  The separate student account workflow links the account atomically.
- No student deletion operation exists. LEFT is a lifecycle status, not a
  delete operation.
- The repository enforces unique NIS/NISN and optional unique user linkage.
  Student records may exist without a login account.

### Read access and scope

- `Get`: Admin may read any student; Wali Kelas must supply an academic year
  and pass `RequireStudentScope`; Siswa may read only the student linked to
  their own user ID.
- `List`: Wali Kelas must use an academic year and may filter by one assigned
  class. With no class filter, the service returns students in all classes
  assigned to that Wali for the year. The HTTP page defaults to the open year.
  Admin can list by class/year or use the all-students and unplaced-students
  views. Siswa is denied list access by this service.
- The list repository orders by student name and ID and supports limit/offset,
  search across name/NIS/NISN, and an optional ACTIVE/LEFT status filter.
  Wali is scoped to assigned classes in the selected year; Admin also has all
  and unplaced master-record views.
- `Get` returns the student master record. `ListClassHistory` separately
  exposes history for a selected year with Admin/Wali/Siswa scope checks. The
  HTTP layer does not query repositories directly.

### Class history and transfer

- `StudentClassHistory` records student, academic year, class, and a half-open
  validity period. Domain/database constraints enforce valid periods and no
  overlap for a student/year.
- `AssignClass` creates a bounded history starting at the configured class
  effective date; existing overlap and academic-year constraints remain
  authoritative.
  `ClassTransferService` continues to handle in-year transfer requests.
- FASE 16.7 decision `DEC-16-04` requires a LEFT student to end active class
  history on the leaving date. `StudentService.Leave` performs status change
  and history closure in one unit of work and blocks if any transfer is
  PENDING. Student/transfer row locking serializes leave against new transfer
  creation.
- FASE 16 audits (`GAP-STUDENT-CLASS-01`) assigned initial placement to FASE
  19; this phase now provides the explicit placement operation.

### Student leaving and financial lifecycle

- `Leave` is Admin-only and allows the domain transition ACTIVE → LEFT. A
  repeated leave or other invalid status transition fails. It closes the
  history containing the supplied date and does not delete the student or
  automatically settle a savings account.
- `Reactivate` is Admin-only and allows LEFT → ACTIVE only when a new class
  history can be created. The effective date must be inside the selected
  academic year and not precede the class's configured effective date. The new
  history starts on that date and runs to the academic-year boundary. Status
  and history are committed in one transaction; the old history is untouched.
- Leaving settlement is a separate service workflow. Its existing contract
  requires a LEFT student and an OPEN savings account. Student HTTP must not
  invent settlement behavior or imply that leaving automatically settles the
  account.

## 2. Roles and routes proposed from existing service behavior

All routes require authentication. Unsafe requests require CSRF protection,
the established request body limit, and POST/303/Redirect behavior used by
FASE 18. The import route accepts multipart form data. The actor must come from
authenticated request context; never accept actor IDs or roles from query/form
data. Hiding controls does not replace service authorization.

| Method | Path | Intended page/action | Existing service operation | Status |
|---|---|---|---|---|
| GET | `/students` | Admin all/unplaced list or scoped class list | `StudentService.List` | Implemented |
| POST | `/students` | Create student | `StudentService.Create` | Available; Admin only |
| POST | `/students/import` | Import validated student master records from `.xlsx` | Student import service operation | Admin only; all-or-nothing |
| GET | `/students/import/template` | Download the `.xlsx` import template | HTTP-generated workbook | Admin only |
| GET | `/students/{id}` | Student detail | `StudentService.Get` | Available; Admin/Wali/Siswa scopes apply |
| GET | `/students/{id}/edit` | Edit form | `StudentService.Get` | Available; Admin only |
| POST | `/students/{id}` | Update student | `StudentService.Update` | Available; Admin only |
| GET | `/students/{id}/class-history` | Class history for selected year | `StudentService.ListClassHistory` | Implemented |
| POST | `/students/{id}/class-assignments` | Initial/cross-year class placement | `StudentService.AssignClass` | Implemented |
| POST | `/students/{id}/leave` | Mark student LEFT and close history | Updated `StudentService.Leave` | Implemented atomically with transfer guard |
| POST | `/students/{id}/reactivate` | Mark student ACTIVE and create new class history | `StudentService.Reactivate` | Admin only; atomic |

Student self-service detail may be reached through the authenticated student
record. This contract does not propose a list route for Siswa because the
existing service rejects it. Wali Kelas list/detail use an academic year;
class is optional on the list and, when omitted, results include all classes
assigned to the Wali in that year. The list defaults to the open academic year.

## 3. Forms and presentation

- Create fields: `name`, `nis`, `nisn`, `academic_year_id`, and `class_id`.
- Update fields: `name`, `nis`, `nisn`, optional `user_id`. Status is not
  editable through the ordinary edit form.
- The form must follow the locked FASE 17 Form UI Pattern and render safe
  validation feedback without exposing repository/database details.
- Wali's list defaults to the open academic year and all classes assigned for
  that year; selecting a class narrows the list. List pagination uses `page`
  (default 1), ten rows per page, and one-row
  lookahead. A legacy `limit` query parameter is validated but does not change
  the shared table size. Wali rows require the
  explicitly selected academic year/class. Admin can also see the master
  student list and students without class history; class/year filtering
  remains available. Search and ACTIVE/LEFT filtering apply to each view.
- The class selector reloads options after an academic year is selected, so
  the user can choose a class in a separate step without a required empty
  class field blocking the request.
- The import workbook has required `name`, `nis`, and `nisn` columns. Optional
  `user_id` is excluded from bulk import; login association remains a separate
  workflow. Reject empty files, unsupported formats, malformed workbooks,
  missing/duplicate headers, blank required values, and identifiers that
  duplicate within the workbook or already exist in the database.
- Accept `.xlsx` only, read the first worksheet, cap uploads at 5 MiB and
  imports at 5,000 student rows, and provide a downloadable workbook template
  with text-formatted columns.
- Import validation reports the affected spreadsheet row numbers. Any error
  rejects the whole file with no persisted students. A fully valid workbook
  is inserted atomically; database conflicts during insert roll back every
  row and return a notification. Each imported student receives the initial
  placement specified by the workbook. No login account or savings account is
  created.
- Successful create and import redirect to the student list with a success
  notice; import includes the imported row count. Update
  redirects with HTTP 303. No delete control or endpoint is exposed.

## 4. Approved decisions and implementation boundaries

- The chosen academic year identifies the history period to place or close.
- Initial placement uses the configured class effective date and existing
  history/database constraints; HTTP must not write history rows directly.
- Leaving requires an active history containing the supplied leave date. If
  no such history exists, the service returns the existing not-found or
  invalid-transition outcome and makes no partial change.
- The pending-transfer guard covers all academic years and must execute inside
  the same unit of work as the status/history change.
- Reactivation requires a new history period with an explicit effective date;
  the previously closed period is never reopened or overwritten.
- Settlement remains separate; the page can link to the account/settlement
  workflow without changing the Leave service's financial behavior.
- The student edit form may associate an existing Siswa user; it does not
  create a login account.
- Bulk import uses the same Student domain validation and uniqueness rules as
  individual creation; the single transaction prevents partial imports.

## 5. Verification required before lock

- Tests passed for scope, self access, validation, pagination,
  search/status filtering, class-history visibility, assignment date bounds,
  leave/history atomicity, transfer-versus-leave serialization, and no-delete
  behavior.
- HTML template rendering and authenticated/CSRF-protected routes passed.
- Admin all/unplaced list behavior, year-dependent class options,
  `.xlsx` parsing, row-level error messages, duplicate detection, and atomic
  rollback on every import failure.
- `make test-integration` passed against the guarded
  `tabungan_siswa_fase168_clean_test` database. This includes migration
  `000007_year_scoped_classes` up/down/up preservation coverage.
- `go test -race ./... -count=1` passed against the same guarded test database.
- `go vet ./...`, `go build ./...`, `npm run test:instant-search`, and
  `git diff --check` passed.
- Automatic Wali year default, all-assigned-classes list behavior, and
  reactivation authorization/date/history/rollback behavior were included in
  the final verification run recorded below.
- Operator guidance requires migration `000007_year_scoped_classes` before
  running the updated application; see the README and placement workflow.

Final verification on 2026-10-19:

- `make test-integration` passed all Go packages against the guarded
  `tabungan_siswa_fase168_clean_test` database.
- `go test -race ./... -count=1` passed against the same disposable database.
- `go vet ./...` and `go build ./...` passed.
- `npm run test:instant-search` passed all seven tests.
- `git diff --check` passed.
- HTTP, service, PostgreSQL integration, and template tests cover student
  placement/reactivation, scope and lifecycle behavior, account creation
  filtering, and transfer request authorization.

The FASE 19 implementation lock is recorded in
[`fase-19-implementation-lock.md`](fase-19-implementation-lock.md).

## 6. Audit sources

- Roadmap FASE 19, page 8: Student CRUD, list search/filter/pagination,
  detail/class history, leaving, class assignment/history, role scopes, and
  locked form pattern.
- `docs/contracts/fase-16-7-decision-resolution.md`: DEC-16-04 and ownership
  by FASE 19.
- `docs/contracts/fase-16-8-implementation-lock.md`: DEC-16-04 remains
  assigned to FASE 19.
- `docs/contracts/fase-16-1-domain-contract-audit.md` and
  `docs/contracts/fase-16-6-contract-audit.md`: initial placement gap
  `GAP-STUDENT-CLASS-01`.
- `docs/contracts/fase-16-4-authorization-contract-audit.md`:
  Admin/Wali/Siswa authorization and student scope.
- Source inspected: `internal/domain/student.go`,
  `internal/domain/student_class_history.go`,
  `internal/service/student_service.go`,
  `internal/service/contracts.go`, `internal/service/class_transfer_service.go`,
  `internal/repository/contracts.go`, and the PostgreSQL student/history
  repositories.
