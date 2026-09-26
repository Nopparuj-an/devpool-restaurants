# Frontend (Next.js)

See the root [README](../README.md) for setup and [docs/](../docs/README.md) for the wiki.

```sh
pnpm install
pnpm dev        # http://localhost:3000; needs the API (make api) and infra (make up)
```

`/api/*` and `/images/*` are rewritten to the Go API and Garage (`next.config.ts`). The rewrite destinations are fixed at build time.
