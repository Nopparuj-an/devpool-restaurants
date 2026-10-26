# Backend

## API

HTTP handler
- [route registrations](backend/internal/server/server.go)

Reservation logic
- [create](backend/internal/reservation/service/reservation_service.go#70)
- [transaction helper](backend/internal/platform/database/database.go#42)
- [peak load](backend/internal/booking/peak.go)

Rating logic
- [upsert](backend/internal/review/service/review_service.go#59)
- [bayesian](backend/internal/restaurant/repository/restaurant_repository.go#184)
- [calculation](backend/internal/platform/database/migrations/00005_rating_triggers.sql)

Timezone
- [opening hours expansion](backend/internal/booking/hours.go#40)
- [tzdata import](backend/cmd/api/main.go#14)
- [timezone validation](backend/internal/restaurant/service/validate.go#55)
- [ADR-0007](docs/decisions/0007-time-storage-and-overnight-hours.md)

## Auth
- [session middleware](backend/internal/auth/middleware.go#23)
- [session cookie](backend/internal/auth/handler.go#173)
- [token hashing](backend/internal/auth/service/auth_service.go#327)
- [Google flow](backend/internal/auth/google_handler.go)
- [ADR-0002](docs/decisions/0002-go-native-auth.md)

# Frontend

React query
- [client setup](frontend/lib/queries.ts)
- [api client](frontend/lib/api-client.ts)
- [infinite list example](frontend/app/_home/container.tsx)

Login and sessions
- email
  - [login/signup submit](frontend/app/_auth/container.tsx#36)
  - [form](frontend/components/app/auth-form.tsx)
  - [signup handler](backend/internal/auth/handler.go#40)
  - [login handler](backend/internal/auth/handler.go#60)
- Google
  - [button](frontend/components/app/auth-form.tsx#57)
  - [start](backend/internal/auth/google_handler.go#53)
  - [callback](backend/internal/auth/google_handler.go#88)
- [/api proxy](frontend/proxy.ts)
- [useAccount](frontend/lib/queries.ts#55)
- [useRequireAccount](frontend/lib/queries.ts#72)
