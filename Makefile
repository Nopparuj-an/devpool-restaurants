# Restaurants — dev tasks. Run `make help`.
COMPOSE := docker compose -f deployment/docker-compose.yml
ENV_FILE := deployment/.env

.PHONY: help env up down ps logs garage-init psql api test fmt vet

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-12s %s\n",$$1,$$2}'

env: ## Create deployment/.env with random secrets (no-op if it exists)
	deployment/make-env.sh

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

api: ## Run the Go API on the host (applies migrations on start)
	cd backend && set -a && . ../$(ENV_FILE) && set +a && \
	  DATABASE_URL="postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@localhost:5432/$$POSTGRES_DB?sslmode=disable" \
	  COOKIE_SECURE=false go run ./cmd/api

test: ## Run backend tests (integration tests use throwaway DBs on the compose Postgres)
	cd backend && set -a && . ../$(ENV_FILE) && set +a && \
	  TEST_DATABASE_URL="postgres://$$POSTGRES_USER:$$POSTGRES_PASSWORD@localhost:5432/postgres?sslmode=disable" \
	  go test ./...

fmt: ## gofmt backend
	cd backend && gofmt -w .

vet: ## go vet backend
	cd backend && go vet ./...
