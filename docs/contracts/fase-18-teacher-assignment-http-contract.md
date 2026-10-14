# FASE 18: Teacher Class Assignment HTTP Contract

Status: IMPLEMENTED IN PATCH; VERIFICATION PENDING

## Scope

This slice provides an Admin-facing full-page HTML workflow for viewing teacher assignments by academic year and creating an assignment. It reuses the existing `ClassAssignmentService`, `UserService`, `ClassService`, and `AcademicYearService`; the HTTP handler does not access repositories directly.

## Routes

- `GET /teacher-assignments`: authenticated page. Only `ADMIN` may use the management page. The optional `academic_year_id` query parameter selects the academic year.
- `POST /teacher-assignments`: authenticated, CSRF-protected form submission. Requires `academic_year_id`, `class_id`, and `teacher_id`; invokes `ClassAssignmentService.Assign` and redirects with HTTP 303 after success.

POST requests use the existing academic-year form-body limit. Forms use `application/x-www-form-urlencoded`. Internal service/repository error details must not be rendered to the user.

## Authorization

Authentication middleware runs before the handler and CSRF middleware protects unsafe methods. The page explicitly restricts management to `ADMIN`; `ClassAssignmentService.Assign` remains the authoritative business authorization boundary. The handler does not introduce assignment edit/delete operations because the current service contract does not provide them.

## Data handling

The page obtains the teacher options via `UserService.ListByRole` for `WALI_KELAS`, and classes/year options via their services. It does not query repositories directly. The page displays username only, never password hash. The current assignment display uses the service's date-scoped `GetCurrentByClass` operation at the academic year's start date.

## Verification required before lock

- Focused route and handler tests.
- `go vet ./...`.
- `git diff --check`.
- `make test-integration` against the database selected by the Makefile safety guard.

This contract is not locked until those checks pass in the project environment and the change is committed.
