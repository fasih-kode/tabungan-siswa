# FASE 18: Class Transfer Request HTTP Contract

Status: VERIFIED — LOCKED
Scope: expose the existing ClassTransferService through server-rendered HTML.

Verification completed on 2026-10-14:
- Focused HTTP and template tests passed.
- `make test-integration` passed all packages; the target verified
  `tabungan_siswa_fase168_clean_test`.
- `go test -race ./... -count=1` passed against the same verified disposable
  database.
- `go vet ./...`, `go build ./...`, and `git diff --check` passed.
- The close/request serialization test verified that `StartClosing` waits for
  an in-flight request write, then rejects closing after that request commits.

This contract reuses the domain and service rules already present. The HTTP
layer parses requests, calls services, and renders safe responses; it does not
implement transfer authorization, lifecycle rules, or class-history changes.

## 1. Existing Business Contract

- Requests start as `PENDING`. Source and destination classes must differ.
- Admin and Wali Kelas may submit a request. Wali Kelas must have the student
  in scope for the selected academic year. Siswa cannot submit requests.
- A request may be created only while its academic year is `OPEN`, and the
  source class must match the student's class history active when requested.
- At most one `PENDING` request may exist per student and academic year.
- Admin may list pending requests and approve or reject them. Admin, a scoped
  Wali Kelas, and the student who owns the request may read it. Student reads
  are self-scoped by the service.
- Approval requires a pending request and an `OPEN` academic year. Its
  `effective_date` must fall within that academic year and within a source-class
  history range for that student. Approval closes that history and creates the
  destination-class history through the academic year's exclusive end date,
  together with the request update in one unit of work.
- The existing service does not constrain `effective_date` relative to the
  request timestamp or current date. The HTTP handler must not add such a
  constraint.
- Rejection requires a nonblank reason, a pending request, and an `OPEN`
  academic year. Rejection does not change class history.

## 2. Approved Lifecycle Decision

**DEC-18-TR-01 — Keep academic years with pending class transfers open**

An academic year with one or more `PENDING` class transfer requests must remain
`OPEN`. Both `StartClosing` and `Close` must return the existing
invalid-transition outcome and leave the year and requests unchanged. Blocking
`StartClosing` keeps requests actionable because transfer approval and rejection
require an `OPEN` year; otherwise a request could strand the year in `CLOSING`.
The product owner selected this rule on 2026-10-14.

This is a lifecycle/service rule, not an HTTP-only validation. Both lifecycle
operations must lock the academic-year row before checking related records;
transfer request writes lock that same row through the database trigger. This
serializes a request creation against `StartClosing`: the request either
commits first and is observed by the guard, or waits and then fails because the
year is no longer `OPEN`. Keep the `Close` guard as a defensive check for
pre-existing or inconsistent `CLOSING` rows. Whether direct database writes
also need this invariant is outside the HTTP contract and must not be silently
assumed.

## 3. Routes

All pages are full-page `html/template` responses. No partial HTMX response is
introduced by this slice.

| Method | Path | Purpose | Service operation |
|---|---|---|---|
| GET | `/students/{student_id}/class-transfer-requests` | List requests for a student and academic year | `ListByStudent` |
| GET | `/class-transfer-requests/pending` | Admin pending-request queue | `ListPending` |
| GET | `/class-transfer-requests/{id}` | View one request | `Get` |
| POST | `/class-transfer-requests` | Submit a request | `Request` |
| POST | `/class-transfer-requests/{id}/approve` | Approve a request | `Approve` |
| POST | `/class-transfer-requests/{id}/reject` | Reject a request | `Reject` |

The student list requires `academic_year_id`. The pending list renders a year
selection form when `academic_year_id` is omitted; a selected year is required
to load rows. Both lists accept `page` (default 1) and `limit` (default 20,
range 1..100); the handler converts these to service `Limit` and `Offset` and
uses the existing lookahead pattern for next-page availability. Repository
ordering remains authoritative.

## 4. Form Input and HTTP Behavior

Create form fields:

- `student_id`
- `academic_year_id`
- `from_class_id`
- `to_class_id`

The current form accepts UUID values for the source and destination class IDs;
the service remains responsible for validating their relationship to the
student and academic year.

Approval form field: `effective_date`, formatted as ISO `YYYY-MM-DD`.

Rejection form field: `rejection_reason`.

After successful create, approve, or reject, redirect with HTTP 303 to the
request detail page. All routes require authentication. Unsafe methods require
CSRF validation, use `application/x-www-form-urlencoded`, and apply the existing
16 KiB form-body limit. Validation errors may render the relevant form with
safe submitted values and user-facing messages. Internal service or repository
details must not be exposed.

The authenticated actor comes only from request context. Actor IDs, roles, and
scope are never accepted from form fields or query parameters. The service
remains the authorization authority; hiding a link is not authorization.

## 5. Error and Access Boundaries

- Invalid path IDs, dates, pagination, or missing required fields are HTTP 400.
- Unauthenticated requests are rejected by authentication middleware.
- Forbidden access and scope violations follow the shared HTTP error mapping.
- Missing requests follow the shared not-found mapping.
- A duplicate pending request follows the existing conflict mapping.
- Invalid state transitions, including a close attempt blocked by pending
  requests, follow the existing shared domain-error mapping.
- Unknown errors render a generic internal-error response without details.

## 6. Verification Before Lock

- Tests for route wiring, authentication, CSRF, role/scope behavior, form
  parsing, UUID/date/pagination validation, redirects, rendering, and safe
  error messages.
- Service tests proving a year with pending transfers cannot start closing or
  close, and a year without pending transfers still follows its existing
  lifecycle conditions.
- Integration coverage for the start-closing/request serialization boundary,
  defensive close guard, and transfer request, approval, and rejection flows.
- `go vet ./...`, `go build ./...`, `git diff --check`, and
  `make test-integration` using its verified disposable database target.

Lifecycle enforcement, the HTTP slice, and verification are complete. This
contract is locked with the FASE 18 implementation checkpoint.
