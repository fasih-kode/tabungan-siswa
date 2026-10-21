# FASE 24: User & Administration Contract

Status: **APPROVED FOR IMPLEMENTATION; IMPLEMENTATION VERIFIED, OWNER LOCK PENDING**
Date: 2026-10-20
Source scope: [official roadmap](../../Roadmap_Resmi_Tabungan_Siswa.pdf)
Security baseline: [AGENTS.md](../../AGENTS.md)

## Approved decisions

- Admin may list, create, update usernames, activate/deactivate, and reset passwords for WALI_KELAS and SISWA accounts.
- ADMIN accounts are provisioned only by the existing guarded bootstrap command. Admin UI and HTTP requests cannot create or change an account into ADMIN.
- An account linked to a student cannot change role. This phase does not implement role reassignment; role is immutable through account administration.
- A user cannot deactivate their own account. ADMIN accounts cannot be deactivated, preserving the bootstrap-provisioned administrator access.
- Deactivation revokes all sessions for that account. Re-activation does not restore revoked sessions.
- Password reset generates a one-time temporary credential, requires a password change at next login for either managed role, and revokes all existing sessions.
- All administration mutations require an authenticated ADMIN, POST, and CSRF validation. Server-side authorization is authoritative.
- Account activation is independent of the linked student's academic lifecycle status; student status rules remain governed by their existing contracts.

## HTTP/UI scope

- Admin-only user directory with WALI_KELAS and SISWA role filters and visible active status.
- Create the existing supported account types, update username, activate/deactivate, and reset password.
- No session listing or per-session revocation UI is included. Sessions are revoked as specified above.

## Acceptance criteria

1. Direct HTTP requests by WALI_KELAS/SISWA cannot read or mutate user administration.
2. Request parameters cannot create ADMIN accounts or change an account's role.
3. Deactivated accounts cannot authenticate or use existing sessions.
4. Deactivation and password reset revoke all sessions atomically with the account/password change.
5. Self-deactivation and ADMIN deactivation are rejected by service-level checks.
6. Username uniqueness, password hashing, CSRF protection, safe error handling, and one-time password reveal remain enforced.
7. Automated tests cover service authorization and account/session lifecycle; PostgreSQL integration tests cover persistence and rollback behavior.

## Out of scope

- Creating or editing ADMIN identities in the web UI.
- Changing roles of existing accounts or reassigning account/student links.
- Listing sessions, revoking an individual session, or changing session expiry duration.
- Auditing account lifecycle events beyond existing application logging. A durable account audit trail requires a separate approved contract.

## Implementation lock

Implementation and automated verification are recorded in [HANDOVER.md](../HANDOVER.md). Owner acceptance, the roadmap-required commit, and the implementation lock record are still pending. Do not treat this contract as a completed FASE 24 lock.
