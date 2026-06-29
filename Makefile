# InterviewOS — root developer Makefile
# Orchestrates the local stack (infra/docker-compose.yml), backend (Go), and frontend (Next.js).

COMPOSE := docker compose -f infra/docker-compose.yml
BACKEND  := backend
FRONTEND := frontend

.DEFAULT_GOAL := help

## ---------- Meta ----------
.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

## ---------- Local stack ----------
.PHONY: dev
dev: ## Boot the full stack (postgres, redis, mailhog, backend, frontend, nginx)
	$(COMPOSE) up --build

.PHONY: up
up: ## Start the stack in the background
	$(COMPOSE) up -d --build

.PHONY: down
down: ## Stop and remove the stack
	$(COMPOSE) down

.PHONY: logs
logs: ## Tail stack logs
	$(COMPOSE) logs -f

.PHONY: ps
ps: ## Show stack status
	$(COMPOSE) ps

## ---------- Database ----------
.PHONY: migrate
migrate: ## Run database migrations
	cd $(BACKEND) && go run ./cmd/migrate

.PHONY: seed
seed: ## Load curriculum seed data (DSA + System Design)
	cd $(BACKEND) && go run ./cmd/seed

## ---------- Backend ----------
.PHONY: be-build
be-build: ## Build the Go backend
	cd $(BACKEND) && go build ./...

.PHONY: be-test
be-test: ## Run backend tests
	cd $(BACKEND) && go test ./...

.PHONY: be-lint
be-lint: ## Vet + lint the backend
	cd $(BACKEND) && go vet ./...

.PHONY: be-tidy
be-tidy: ## Tidy Go modules
	cd $(BACKEND) && go mod tidy

## ---------- Frontend ----------
.PHONY: fe-install
fe-install: ## Install frontend deps
	cd $(FRONTEND) && npm install

.PHONY: fe-build
fe-build: ## Build the frontend
	cd $(FRONTEND) && npm run build

.PHONY: fe-typecheck
fe-typecheck: ## Typecheck the frontend
	cd $(FRONTEND) && npm run typecheck

.PHONY: fe-lint
fe-lint: ## Lint the frontend
	cd $(FRONTEND) && npm run lint

## ---------- Aggregate ----------
.PHONY: build
build: be-build fe-build ## Build backend + frontend

.PHONY: test
test: be-test ## Run all test suites

.PHONY: lint
lint: be-lint fe-lint ## Lint backend + frontend
