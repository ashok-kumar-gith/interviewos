# InterviewOS — root developer Makefile
# Orchestrates the local stack (infra/docker-compose.yml), backend (Go), and frontend (Next.js).

COMPOSE := docker compose -f infra/docker-compose.yml
BACKEND  := backend
FRONTEND := frontend

# Local backend base URL (sudo-free dev stack and docker-compose both expose :8080).
API_URL ?= http://localhost:8080
# Helm release name + namespace for the k8s convenience targets.
HELM_RELEASE ?= interviewos
K8S_NAMESPACE ?= interviewos
HELM_CHART := infra/helm/interviewos

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
be-test: ## Run backend unit tests
	cd $(BACKEND) && go test ./...

.PHONY: be-test-integration
be-test-integration: ## Run backend tests serially (-p 1) against a real Postgres+Redis
	cd $(BACKEND) && go test ./... -p 1

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

## ---------- Observability / API ----------
.PHONY: metrics
metrics: ## Fetch Prometheus metrics from the running backend
	@curl -sf $(API_URL)/metrics || echo "backend not reachable at $(API_URL)"

.PHONY: swagger
swagger: ## Open the live Swagger UI (served by the backend)
	@echo "Swagger UI: $(API_URL)/swagger"
	@(command -v open >/dev/null && open $(API_URL)/swagger) || \
	 (command -v xdg-open >/dev/null && xdg-open $(API_URL)/swagger) || true

## ---------- Kubernetes / Helm ----------
.PHONY: k8s-apply
k8s-apply: ## Apply the raw K8s manifests (kubectl apply -k infra/k8s)
	kubectl apply -k infra/k8s

.PHONY: k8s-delete
k8s-delete: ## Delete the raw K8s manifests
	kubectl delete -k infra/k8s

.PHONY: k8s-dryrun
k8s-dryrun: ## Client-side dry-run of the K8s manifests
	kubectl apply --dry-run=client -k infra/k8s

.PHONY: helm-template
helm-template: ## Render the Helm chart to stdout (no cluster needed)
	helm template $(HELM_RELEASE) $(HELM_CHART)

.PHONY: helm-install
helm-install: ## Install/upgrade the Helm chart into the cluster
	helm upgrade --install $(HELM_RELEASE) $(HELM_CHART) \
		--namespace $(K8S_NAMESPACE) --create-namespace

.PHONY: helm-uninstall
helm-uninstall: ## Uninstall the Helm release
	helm uninstall $(HELM_RELEASE) --namespace $(K8S_NAMESPACE)
