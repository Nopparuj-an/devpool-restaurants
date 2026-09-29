# Architecture Decision Records

One file per decision: `NNNN-short-title.md`. Each has a Status (`Proposed` | `Accepted` | `Superseded by NNNN`) plus Context, Decision, and Consequences sections. Never edit an accepted ADR's decision. Supersede it with a new one instead.

| # | Decision | Status |
|---|---|---|
| [0001](0001-15-minute-slots.md) | Reservations use a 15-minute grid | Accepted |
| [0002](0002-go-native-auth.md) | Go-native auth (email/password + Google), no Keycloak | Accepted |
| [0003](0003-booking-concurrency.md) | Per-restaurant row lock for booking writes | Accepted |
| [0004](0004-rating-aggregate-and-ranking.md) | Stored rating aggregate + Bayesian ranking | Accepted (upkeep superseded by 0015) |
| [0005](0005-garage-object-storage.md) | Garage (S3) for images | Accepted |
| [0006](0006-restaurant-changes-and-deletion.md) | Owner edits don't touch bookings; delete cascades | Accepted |
| [0007](0007-time-storage-and-overnight-hours.md) | UTC everywhere, browser-local display, overnight/24h shifts | Accepted |
| [0008](0008-design-as-react-components.md) | UI design as in-repo React components, not Claude Design/Lovable | Accepted (risk noted) |
| [0009](0009-dev-stack-and-local-infra.md) | Dev stack & local infra (stdlib router, pgx, goose, compose bind mounts) | Accepted |
| [0010](0010-containers.md) | API/web images; app compose separate from infra | Accepted |
| [0011](0011-gin-hexagonal-layout.md) | Gin + hexagonal per-feature layout (like cpc-be) | Accepted |
| [0012](0012-admin-bans.md) | Admin rights in the DB; reversible bans; ratings rebuilt on ban | Accepted (ratings part superseded by 0015) |
| [0013](0013-client-side-data-loading.md) | Data loads in the browser with TanStack Query; `/api` proxied, cookie unchanged | Accepted |
| [0014](0014-admin-impersonation.md) | Admin impersonation as a marked session, one cookie, dies with admin rights | Accepted |
| [0015](0015-rating-totals-by-trigger.md) | Rating totals recomputed by DB triggers, CHECK-guarded | Accepted |
