# 0013 — Client-side data loading with TanStack Query

**Status:** Accepted (2026-09-26)

## Context
Pages loaded their data in Server Components, which forwarded the session cookie to Go. It worked, but every navigation came back as an RSC payload (`?_rsc=`), so the network tab never showed the API calls, and the frontend was a server app. The team's usual setup is a separate frontend and backend with a client-side app, and that's what we want to explain in the interview.

## Decision
- **All data loads in the browser.** Route components in `frontend/components/pages/pages.tsx` read the URL and use TanStack Query (`useQuery`, `useInfiniteQuery` for "Show more"). `app/**/page.tsx` only sets the title and renders one of them. The screens are unchanged: they still take data as props, and `/design` still feeds them mock data.
- **The session stays an HttpOnly cookie.** No JWT, no token in JavaScript. The browser calls `/api/*` on the web origin, and a path-based proxy forwards it to Go, so the cookie is first-party and `SameSite=Lax` with no CORS. Today the proxy is the Next server's rewrite (`next.config.ts`), which also serves `/images/*` from Garage. Any reverse proxy (Caddy, nginx) that routes `/api` and `/images` the same way can replace it.
- **After a write, invalidate everything** (`useRefresh()` in `lib/queries.ts`, which calls `invalidateQueries()`). Only queries on screen refetch; the rest refetch when next used. That replaces `router.refresh()` and keeps writes simple. Per-key invalidation would save a few requests but adds a way to show stale data.
- **Guards are UX, not security.** `useRequireAccount` sends logged-out visitors to `/login?next=…`; owner and admin pages show the 404 page to others. The API enforces every rule anyway (R-REST-2, R-ADMIN-1).
- A 401 from any read sets the cached account to `null`, so the header shows the visitor as logged out.

## Consequences
- The network tab shows real `/api/*` requests. Page navigations still fetch a small RSC payload for the route shell; that's how the App Router navigates.
- Routes without an `[id]` are prerendered as static shells. The first view shows a spinner until the data arrives, and restaurant pages have no server-rendered content for search engines.
- Anonymous visitors make one `GET /api/me` that returns 401 (the browser logs it). JavaScript can't see an HttpOnly cookie, so it can't skip the call.
- Moving to `output: "export"` later only needs: `[id]` routes changed to query strings (`/restaurant?id=1`), and something other than Next to serve the files and proxy `/api` and `/images` (Go with `go:embed`, or Caddy).
