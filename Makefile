# Restaurants — dev tasks. Run `make help`.
COMPOSE := docker compose -f deployment/docker-compose.yml
APP_COMPOSE := docker compose -f deployment/docker-compose.app.yml
ENV_FILE := deployment/.env

# Env for running Go commands on the host against the compose infra.
HOST_ENV = set -a && . ../$(ENV_FILE) && set +a && \
	  export DATABASE_URL="postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@localhost:5432/$$POSTGRES_DB?sslmode=disable" \
	  S3_ENDPOINT=http://localhost:3900 COOKIE_SECURE=false

.PHONY: help env up down ps logs garage-init psql api seed seed-bulk seed-bulk-remove test fmt vet app-up app-down app-logs app-seed

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-12s %s\n",$$1,$$2}'

# DEV

api: ## Run the Go API on the host (applies migrations on start)
	cd backend && $(HOST_ENV) && go run ./cmd/api

fe: ## Start frontend dev
	cd frontend && pnpm dev

test: ## Run backend tests (integration tests use throwaway DBs on the compose Postgres)
	cd backend && set -a && . ../$(ENV_FILE) && set +a && \
	  TEST_DATABASE_URL="postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@localhost:5432/postgres?sslmode=disable" \
	  TEST_S3_ENDPOINT=http://localhost:3900 \
	  TEST_IMAGE_PUBLIC_URL=http://$$S3_BUCKET.web.garage.localhost:3902 \
	  go test ./...

fmt: ## gofmt backend
	cd backend && gofmt -w .

vet: ## go vet backend
	cd backend && go vet ./...

# PREPARE

env: ## Create deployment/.env with random secrets (no-op if it exists)
	deployment/make-env.sh

seed: ## Load demo accounts, restaurants, bookings, reviews (no-op if already seeded)
	cd backend && $(HOST_ENV) && go run ./cmd/seed

N ?= 1000
seed-bulk: ## Add N generated users + restaurants + ~5N reviews for load checks (N=1000)
	cd backend && $(HOST_ENV) && go run ./cmd/seed -bulk $(N)

seed-bulk-remove: ## Remove the generated bulk data (and its photos)
	cd backend && $(HOST_ENV) && go run ./cmd/seed -bulk-remove

# INFRA

up: ## Start Postgres + Garage
	@mkdir -p deployment/data/postgres deployment/data/garage/meta deployment/data/garage/data
	$(COMPOSE) up -d --wait

down: ## Stop infra (data in deployment/data is kept)
	$(COMPOSE) down

ps: ## Show infra status
	$(COMPOSE) ps

logs: ## Tail infra logs
	$(COMPOSE) logs -f

garage-init: ## One-time Garage layout/key/bucket setup (idempotent)
	deployment/garage-init.sh

psql: ## Open psql in the Postgres container
	$(COMPOSE) exec postgres sh -c 'psql -U $$POSTGRES_USER -d $$POSTGRES_DB'

app-up: ## Build + start API and web containers (needs `make up`); web on :3000
	$(APP_COMPOSE) up -d --build --wait

app-down: ## Stop API and web containers
	$(APP_COMPOSE) down

app-logs: ## Tail API and web logs
	$(APP_COMPOSE) logs -f

app-seed: ## Load demo data using the API image (no-op if already seeded)
	$(APP_COMPOSE) run --rm --no-deps --entrypoint /seed api
