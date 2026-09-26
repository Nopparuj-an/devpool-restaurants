# Roadmap

Deadline: **2026-10-11** (submit). Interview: 2026-10-17/18. Tick items as they land.

## M0 — Foundation (26–28 Sep)
- [ ] UI design in Claude Design: list, detail + booking, my reservations, my restaurants (CRUD), owner bookings, login
- [ ] `deployment/docker-compose.yml`: Postgres, Garage (+ init script), API, web
- [ ] Go skeleton: config, router, migrations, error format (see [architecture](architecture.md#api-conventions))
- [ ] Auth: password signup/login/logout, sessions, `me` endpoint
- [ ] Next.js skeleton: `/api` rewrite, session-aware layout

## M1 — Restaurants (29 Sep–1 Oct)
- [ ] Restaurant CRUD with ownership checks (R-REST-*)
- [ ] Opening hours per weekday incl. overnight + 24h, merge logic (R-HOURS-*)
- [ ] Max reservation duration + cancel cutoff settings (R-REST-3, R-REST-6)
- [ ] Image upload to Garage, cover + extras
- [ ] Public list + detail pages

## M2 — Booking engine (1–4 Oct) ← most important
- [ ] `peakLoad()` pure func + table tests (brief examples, back-to-back, edit excludes self)
- [ ] Shift resolution + overnight tests
- [ ] Create / edit / cancel with row lock (ADR-0003), cutoff rules, 30-day horizon, shrink exception (R-EDIT-3)
- [ ] Concurrency integration test (N goroutines, exactly one wins)
- [ ] My reservations page; booking form with 15-min pickers

## M3 — Reviews (5–6 Oct)
- [ ] Upsert / edit / delete review, owner forbidden, verified badge
- [ ] Aggregate columns + Bayesian sort (ADR-0004); Highest rated / Most reviewed

## M4 — Polish & extras (7–9 Oct)
- [ ] Google OAuth (ADR-0002) incl. password drop on link + set-new-password — after password auth is solid
- [ ] Owner reservation table (per day/slot, full customer info — R-PRIV-2)
- [ ] Limited Seats Left badge
- [ ] Search/filter by name, cuisine
- [ ] Stretch: holidays/closures, search by free time
- [ ] Seed data + test accounts; README (run, accounts, env vars)

## M5 — Submit (10–11 Oct)
- [ ] Record ≤5 min video: demo all features + walk through `peakLoad` + booking tx
- [ ] Final README check on a fresh clone
- [ ] Submit repo + video + design link
