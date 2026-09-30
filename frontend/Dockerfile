# syntax=docker/dockerfile:1
# Next.js web app, built as a standalone server (output: "standalone").

FROM node:24-alpine AS base
ENV NEXT_TELEMETRY_DISABLED=1
# pnpm at the version pinned in package.json#packageManager
RUN corepack enable
WORKDIR /app

FROM base AS deps
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN --mount=type=cache,id=pnpm,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile

FROM base AS build
COPY --from=deps /app/node_modules ./node_modules
COPY . .
# webpack, not Turbopack: Turbopack's production build was OOM-killed in a
# 2 GB Docker VM (Rancher Desktop default); webpack needs far less memory.
RUN pnpm exec next build --webpack

FROM node:24-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 PORT=3000 HOSTNAME=0.0.0.0
# Read per request by proxy.ts; override at `docker run` / in compose.
ENV API_URL=http://api:8080 \
    IMAGES_URL=http://restaurant-images.web.garage.localhost:3902
COPY --from=build --chown=node:node /app/public ./public
COPY --from=build --chown=node:node /app/.next/standalone ./
COPY --from=build --chown=node:node /app/.next/static ./.next/static
USER node
EXPOSE 3000
HEALTHCHECK --interval=10s --timeout=4s --start-period=15s \
  CMD ["node", "-e", "fetch('http://127.0.0.1:3000/').then(r => process.exit(r.ok ? 0 : 1), () => process.exit(1))"]
CMD ["node", "server.js"]
