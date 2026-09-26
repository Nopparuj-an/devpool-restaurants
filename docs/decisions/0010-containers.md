# 0010 — Container images and a separate app compose file

**Status:** Accepted (2026-09-26)

## Decision
- **Two compose files, two projects:**
  - `deployment/docker-compose.yml` (project `restaurants`) holds the infra: Postgres and Garage, with data bind-mounted in `deployment/data/`.
  - `deployment/docker-compose.app.yml` (project `restaurants-app`) holds the app: the Go API and Next.js web. It joins the infra network (`restaurants_default`) as an **external** network, so `make app-down` never touches the database, and the infra can keep running while the app is rebuilt.
- **API image** (`backend/Dockerfile`): static `CGO_ENABLED=0` binaries on `distroless/static:nonroot` (≈43 MB). It also contains `/seed` (`make app-seed`). Distroless has no curl, so `api healthcheck` is a subcommand that the container healthcheck calls.
- **Web image** (`frontend/Dockerfile`): Next.js `output: "standalone"` on `node:24-alpine`, running as a non-root user. The `/api` and `/images` rewrite destinations are **build args**, because Next serializes rewrites at build time. Server-side fetches read `API_URL` at runtime.
- **The web image is built with webpack (`next build --webpack`), not Turbopack.** Turbopack's production build was OOM-killed in Rancher Desktop's default 2 GB VM, while webpack builds in about 5 s. Dev (`pnpm dev`) still uses Turbopack.
- Only the web app is published (`127.0.0.1:3000`). The API is reached through the `/api` rewrite, the same origin as the browser (docs/architecture.md).

## Two ways to run
| | Infra | API | Web |
|---|---|---|---|
| Host dev | `make up` | `make api` (:8080) | `cd frontend && pnpm dev` (:3000) |
| Containers | `make up` | `make app-up` (both, :3000) | |

Both use port 3000, so run one or the other.

## Consequences
- Verified on 2026-09-26: SSR through the API, cookie login through the rewrite, images through the rewrite to Garage (the Host header reaches the right bucket), and a 3.6 MB multipart upload through the proxy.
- `COOKIE_SECURE` is `false` in the app compose because it's plain http on localhost. Set it to `true` behind HTTPS.
