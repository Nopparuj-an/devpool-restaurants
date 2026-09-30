# 0016 — Images built in GitHub Actions, deployed by Watchtower

**Status:** Accepted (2026-09-30)

## Decision
- `.github/workflows/publish.yml` builds `backend/` and `frontend/` on every push to `main` and every `v*` tag, and pushes them to GHCR as `ghcr.io/<owner>/restaurants-api` and `restaurants-web`. Tags: `latest` (main), `sha-<short>`, and the semver version on tags.
- `deployment/docker-compose.prod.yml` is a self-contained production stack: `restaurants-web`, `restaurants-api`, `restaurants-postgres` (`postgres:16-alpine`) and `restaurants-garage`, with all state in named volumes, so `docker compose down -v` removes everything. It reuses an existing Traefik (labels; one host, `restaurants.anan.ovh`, for the web app). Watchtower is opt-in (`--profile watchtower`) and only updates containers labelled `com.centurylinklabs.watchtower.enable`.
- **Networks:** the external `proxy` network (Traefik's, named exactly `proxy`) and a private `internal` network. Only web joins `proxy`; API, Postgres and Garage are on `internal` only. Services carry a `restaurants-` prefix because web's name is registered on the shared `proxy` network.
- **Seeding:** `make prod-seed` runs the API image's `/seed` binary as a one-off container of the `restaurants-api` service (same env and network), so it reaches Postgres and Garage by name.
- **Hardening:** every container drops all capabilities, sets `no-new-privileges`, has a read-only root filesystem (tmpfs only for `/tmp` and Next's cache), and PID and memory limits; no ports are published. The API gets only the variables it reads, not the whole `.env`. API and web run as their images' non-root users. Postgres runs as uid 70 (the entrypoint's chown/gosu step needs capabilities). Garage's image has no non-root user, so it runs as root without capabilities.
- **The images take no build-time configuration**, so anyone can pull and run them. The web app proxies `/api` and `/images` from `frontend/proxy.ts`, which reads `API_URL` and `IMAGES_URL` from the environment on each request. This replaces the build-time `rewrites()` (and the build args) of ADR-0010. The image defaults (`http://api:8080`, `http://restaurant-images.web.garage.localhost:3902`) fit the compose files; override them in the container's `environment:`.
- Only the web host is public. `/api` and `/images` are proxied by the web app over the internal network, which keeps the session cookie same-origin. The API and Garage are not reachable from outside the stack.

## Consequences
- `proxy.ts` buffers request bodies. `experimental.proxyClientMaxBodySize` is raised to 110 mb in `next.config.ts` so multi-image uploads (10 x 10 MB) are not truncated at the 10 MB default. Keep it above `maxRequestBytes` in the backend's `image_handler.go`.
- Publishing is not gated on CI: a commit that fails CI is still published. Gate it with `workflow_run` if that matters.
- The API applies migrations on start, so a Watchtower restart migrates the database.
- Images are `linux/amd64` only.
- The Watchtower container needs the Docker socket, which is root-equivalent on the host.
- Verified locally (2026-09-30) with the hardening on: health checks pass, the web app reaches the API and Garage, and writes outside the tmpfs mounts fail. The Traefik labels were not exercised (no Traefik in the test).
- GHCR packages are private by default: make them public, or `docker login ghcr.io` on the server.
