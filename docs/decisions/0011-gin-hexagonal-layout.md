# 0011 — Gin and a hexagonal, per-feature layout

**Status:** Accepted (2026-09-26). Supersedes the "stdlib `net/http` routing" part of ADR-0009.

## Context
The backend started as one package per feature, with handlers, SQL and rules side by side, on stdlib routing. It worked, but finding "where is X" meant reading code. The team's other Go project (`cpc-be/internal/dashboard`) uses Gin with a hexagonal layout, and matching it makes both projects look alike.

## Decision
- **Gin** for routing and middleware.
- **Every feature has the same shape**: `routes.go`, `handler.go`, `module.go`, `model/`, `service/`, `repository/`. The service declares its inbound port (`Service`) and outbound ports (`Repository`, and `ImageStore` for restaurants) as interfaces. Repositories are the SQL adapters. See [backend.md](../backend.md).
- **Transactions live in services** (`WithinTx`). Repositories join them via the context (`db.Conn(ctx)`), like `database.FromContext` in cpc-be.
- **Errors carry a Kind, not a status** (`platform/apperr`). `platform/web` maps a Kind to a status code, so services never import Gin.
- Pure rules stay in `internal/booking`, shared by the reservation service and the restaurant repository.

### Differences from cpc-be, on purpose
- **No fx (dependency injection container).** Each `module.go` wires its own feature, and `server.New` composes them in about ten lines that can be read top to bottom. That's easier to explain in the interview than container lifecycles.
- **pgx and hand-written SQL, not GORM** (ADR-0009). The booking lock (`SELECT … FOR UPDATE`) and the Bayesian sort are the parts the interview asks about, and they read best as plain SQL.

## Consequences
- More files, less to read per file. A new endpoint touches `routes.go`, `handler.go`, the service, maybe the repository.
- The API contract didn't change. The API tests (`<feature>_test.go`) are unchanged apart from import paths and all pass, as does `pnpm e2e`. `TestBookingWaitsForRestaurantLock` still fails when `FOR UPDATE` is removed (checked after the move).
