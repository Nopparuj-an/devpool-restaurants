# 0002 — Go-native auth: email/password + Google OAuth, no Keycloak

**Status:** Accepted (2026-09-26)

## Context
Keycloak was first considered. It is a lot of infrastructure for a 2-week exam, and the interview asks "where is login/session stored?". A flow we wrote ourselves is easier to explain than a black box.

## Decision
- The Go API owns auth. Two identity types: `password` (argon2id or bcrypt hash) and `google` (OIDC authorization-code flow, verifying the ID token with Google's JWKS, e.g. via `coreos/go-oidc` + `golang.org/x/oauth2`).
- Sessions are server-side in Postgres. The cookie holds an opaque random token, and the DB stores its SHA-256 hash. Cookie flags: `HttpOnly; Secure; SameSite=Lax; Path=/`. Sessions are revocable on logout.
- **Trusted Google email:** a Google login is accepted only if `email_verified=true`. If an account with that email exists, the Google identity is linked to it.

## Account pre-hijack protection
Risk: an attacker signs up with a password using the victim's email before the victim ever logs in. The victim later logs in with Google, gets linked into the attacker's account, and the attacker still knows the password.

Mitigation (decided 2026-09-26): **Google always takes priority.** No email connector is planned, so password signups are never email-verified. On the first Google login that links to an existing account:
1. Delete the account's `password` identity and **all its sessions**.
2. Set `accounts.email_verified = true`.
3. The user can set a new password from account settings while logged in via Google. That creates a fresh `password` identity, trusted because the email is now verified.

Later Google logins to an already verified account just log in.

## Consequences
- No external IdP container. Fewer moving parts in docker compose.
- We own the security details: password hashing, the OAuth `state`/PKCE check, session expiry, and login rate limiting (nice to have).
- Google OAuth needs a client ID/secret and a redirect URI registered in Google Cloud Console. The README must explain how to run without it (password login still works, and seed accounts use passwords).
