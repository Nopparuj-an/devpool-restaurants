# 0014 — Admin impersonation

**Status:** Accepted (2026-09-26)

## Context
For demos and support, an admin wants to see the app exactly as a given user does and act for them, then return to their own account without logging in again.

## Decision
- **A second kind of session, not a second cookie.** `sessions.impersonator_id` marks a session an admin opened as another user. `POST /api/admin/users/:id/impersonate` deletes the admin's session and sets a cookie for a new one as the user, recording the admin. `POST /api/auth/impersonate/stop` deletes it and logs the admin in with a fresh session. The browser only ever holds one HttpOnly cookie, and tokens are still stored hashed. The admin's old token can't be restored from its hash, so "back" makes a new one.
- **It lives in the auth module**, because it creates sessions and sets the cookie. The route is under `/api/admin/...` and behind `RequireAdmin`.
- **Limits (R-ADMIN-7):** no admins (no privilege games between admins) and no banned users. Only one level: while impersonating you're not an admin, so you can't start another. The password can't be changed through an impersonation session. The session lasts 8 hours instead of 30 days.
- **It dies with the admin's rights.** The session query only accepts an impersonation session while the impersonator is an admin and not banned, so revoking admin rights ends it at once.
- `/api/me` returns `impersonator: {id, display_name}`, and the header shows a banner with "Back to …".
- Start and stop are logged with slog (admin ID, user ID). There's no audit table yet.

## Consequences
- Anything done while impersonating is recorded as the user's. Only the log says an admin did it.
- If the admin logs out instead of going back, the impersonation session is simply deleted, and they log in again as usual.
- Tests: `backend/internal/admin/manage_test.go`.
