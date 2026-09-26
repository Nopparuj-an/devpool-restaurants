# syntax=docker/dockerfile:1
# Go API (+ seed tool). Static binaries on distroless, running as non-root.

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

# static-debian12 ships CA certificates (Google OIDC over HTTPS); tz data is
# embedded in the binaries via time/tzdata.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /out/seed /
USER nonroot
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=4s --start-period=10s CMD ["/api", "healthcheck"]
ENTRYPOINT ["/api"]
