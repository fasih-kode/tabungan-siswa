# FASE 17.3 — HTTP Acceptance Review

**Status:** REVIEWED — PENDING FINAL VERIFICATION AND COMMIT
**Contract:** `docs/contracts/fase-17-2-http-contract.md`
**Reviewed snapshot:** `21aa0d7 test(auth): cover invalid cookie logout behavior`
**Scope:** Verify the applicable FASE 17 HTTP contract criteria against the source and tests in the reviewed repository snapshot.

This review does not declare FASE 17 LOCK. Final verification after this documentation change and a commit reference are still required.

## 1. Acceptance Matrix

| Contract criterion | Result | Source/test evidence and rationale |
|---|---|---|
| Handler uses service and avoids direct repository access | PASS | `internal/http/authentication_handler.go` depends on `service.AuthenticationService`; session creation and logout are service operations. The dashboard page only prepares a view model and renders a template. |
| Required dependencies are validated during wiring | PASS | Constructors validate required dependencies in `internal/http/authentication_handler.go`, `internal/http/login_page.go`, `internal/http/dashboard_page.go`, `internal/http/handler_set.go`, and router/middleware wiring. Related tests include `TestNewAuthenticationHandlerRejectsMissingDependencies`, `TestNewHandlerSetRejectsMissingAuthenticationService`, and `TestNewRouterRejectsMissingDependencies`. |
| HTTP error mapping is consistent and does not expose internal details | PASS | `internal/http/httperror/http_error.go`, `internal/http/autherror/authentication_error.go`; tests in `internal/http/httperror/http_error_test.go`, `internal/http/autherror/authentication_error_test.go`, and `internal/http/authentication_handler_test.go`. General and authentication mappings remain separate as the contract requires. |
| Form POST/Redirect/GET behavior is tested | PASS | Login redirects with 303 to `/protected`; logout redirects with 303 to `/login` after successful revocation and cookie clearing. Tests cover logout without a cookie, with an invalid cookie, and service failure. See `internal/http/authentication_handler_test.go` and `internal/http/authentication_integration_test.go`. |
| Query parameter parsing and pagination are tested on applicable endpoints | NOT APPLICABLE IN FASE 17 | The reviewed `internal/http` source contains no list endpoint or handler that parses pagination, search, or filter query parameters. Contract sections 7–9 deliberately require these contracts to be set per feature. Do not add generic parsing without a feature endpoint and service contract. Reassess when the first applicable list endpoint is introduced. |
| Presentation follows page needs and does not assume flash-message support | PASS FOR CURRENT PAGES; GENERAL FLASH MESSAGES DEFERRED | `internal/http/login_page.go` and `internal/http/dashboard_page.go` use page-specific view models. No general flash-message mechanism is assumed or added. Form-specific validation presentation is to be specified with the relevant feature rather than invented for pages that do not yet exist. |
| Full-page HTML works without HTMX | PASS FOR CURRENT PAGE FOUNDATION | The login and dashboard handlers render Go `html/template` templates (`internal/http/login_page.go`, `internal/http/dashboard_page.go`). Broad HTMX interactions are outside this phase. |
| Partial responses have explicit endpoint contracts | NOT APPLICABLE IN FASE 17 | No partial-response endpoint is implemented in the reviewed HTTP foundation. Contract section 11 explicitly assigns broad HTMX interaction implementation to FASE 26. A partial response must not be added before a feature endpoint defines it. |
| Relevant focused tests pass | PREVIOUSLY REPORTED PASS; RE-RUN REQUIRED FOR FINAL VERIFICATION | The latest recorded user run passed `go test ./internal/http -count=1`. The reviewed snapshot includes explicit tests for invalid-cookie logout and method handling. Rerun the verification commands in section 3 before committing this review. |
| Formatting, diff checks, vet, and full tests pass | PREVIOUSLY REPORTED PASS; RE-RUN REQUIRED FOR FINAL VERIFICATION | The latest recorded user run passed `RUN_INTEGRATION_TESTS=1 go test ./... -count=1`, `go vet ./...`, and `git diff --check` before this documentation-only review. Rerun them after applying this review. |
| No unrelated domain/feature changes | PASS BY SCOPE REVIEW | The changes under review for this finalization are documentation-only. No domain contract or feature implementation is proposed. |
| Commit exists before FASE 17 LOCK | PENDING | The implementation commits exist through `21aa0d7`; this acceptance review and contract status update must still be committed before a separate lock record can declare the phase locked. |

## 2. Review Findings

1. The invalid-cookie logout case is implemented and explicitly tested. It clears the cookie, redirects with 303 to `/login`, and does not call the logout service without a valid token.
2. Logout service failure does not return a success redirect or clear the cookie.
3. The authentication handler uses the service boundary for session creation and logout rather than directly accessing a repository.
4. Query parameter, pagination, search/filter, general flash messages, and partial responses are not silently treated as implemented. They remain inapplicable to the current HTTP foundation because the corresponding feature endpoints/contracts do not exist in this phase; partial HTMX interaction is explicitly deferred to FASE 26.
5. No source-code change is required by this review. The remaining work is final local verification, committing the finalized contract/review, and recording the commit reference in a dedicated phase-lock document before declaring FASE 17 LOCK.

## 3. Final Verification Required

Run from the project root, with the approved test database configuration used by the project. Do not point integration tests at the development database.

```bash
set -e

unformatted="$(gofmt -l $(find cmd internal -name '*.go' -type f))"
if [[ -n "$unformatted" ]]; then
  printf '%s\n' "$unformatted"
  exit 1
fi

go test ./internal/http -count=1
go vet ./...
git diff --check
make test-integration
```

Use `make test-integration` as the standard full integration-test command. It selects `tabungan_siswa_fase168_clean_test`, verifies the actual connected database before running tests, and never uses the development DSN as `DATABASE_URL`. Do not bypass or weaken the existing Go database guard.

After verification, commit the contract and this review. Only then create/update the phase-lock record with the final commit hash and mark FASE 17 LOCK. Do not claim LOCK before that commit and evidence exist.
