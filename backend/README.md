# Backend (Go + Gin)

Hexagonal, one folder per feature. **The full map is in [docs/backend.md](../docs/backend.md)**: where everything is, how a request flows, which file enforces which rule.

```
cmd/api          server entry point
cmd/seed         demo data
internal/
  server/        composition root (all routes plugged in here)
  booking/       pure seat and time rules, no I/O
  auth/          ─┐
  restaurant/     │ each: routes.go → handler.go → service/ → repository/
  reservation/    │       + model/ (types, errors) + module.go (wiring)
  review/        ─┘
  platform/      apperr, web (Gin helpers), database, storage, config
  testutil/      API test harness, throwaway test databases
```

Run it with `make api`, and test it with `make test` from the repo root.
