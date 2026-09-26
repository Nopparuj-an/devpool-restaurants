# Requirements

## Dates

| Date | What |
|---|---|
| 2026-09-23 | Brief released |
| **2026-10-11** | Submit: repo + video + UI design |
| 2026-10-17 / 18 | Technical interview, 15 min each |

## Tech constraints

| Area | Rule |
|---|---|
| Frontend | **Next.js** (required), for both owner and customer screens |
| Backend | **Go** (required). Booking rules must be checked in Go. The frontend may check them too, but never only there. |
| UI design | **Claude Design** or Lovable (required). Design first, then build in Next.js. ⚠️ We use in-repo React components instead, see [ADR-0008](decisions/0008-design-as-react-components.md) |
| DB, login, libraries | Free choice, but we must be able to explain why |
| Docker, CI/CD | Optional. The app must run, and the README must explain how |

## Features (required)

### 1. Login and restaurant management
- Sign up and log in. One account is both owner (of its own restaurants) and customer (of any restaurant).
- Create, edit, and delete a restaurant: name, details (cuisine type, location, …), at least one image, seat count, opening hours.
- Only the owner can edit or delete a restaurant. **This must be enforced by the API, not just by hiding buttons.**
- Owner sets the cancellation cutoff.

### 2. Reservations
- Customer picks a restaurant and enters party size (pax), date, start and end time.
- See the [booking rules](domain.md#reservations). In short: within opening hours, not in the past, and seats never exceeded at any moment.
- "My reservations" page: list, edit, and cancel your own bookings.

### 3. Ratings and reviews
- 1–5 integer stars plus a text review. Owners can't review their own restaurant.
- Show the average with one decimal, plus the review count.
- Sort restaurants by **Highest rated** or **Most reviewed**.

## Extras (optional, our choice)

Suggestions from the brief: Limited Seats Left badge, owner view of the restaurant's bookings, holidays and per-day hours, search and filters (name, cuisine, free time), photo gallery. See [roadmap.md](roadmap.md) for which ones we plan to do.

## Deliverables

1. **Git repo**: Next.js + Go code, plus a README with run instructions, **test accounts**, and **seed data** so restaurants and bookings show up right away.
2. **Video**, 5 minutes max: demo every feature and explain the key code, especially the seat-capacity check.
3. **UI design**: Claude Design link or screenshots. We submit screenshots of `design/` ([ADR-0008](decisions/0008-design-as-react-components.md)).

## Interview format

5 min: they watch the video. 5 min: technical Q&A on Next.js and Go. 5 min: the panel discusses. The focus is **foundation**: understanding your own code and why it was built that way. See [interview-prep.md](interview-prep.md).
