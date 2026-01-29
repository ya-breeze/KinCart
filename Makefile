.PHONY: build test test-e2e lint docker-up docker-down docker-up-e2e docker-down-e2e add-family add-user help

# Default target
help:
	@echo "KinCart Makefile Tasks:"
	@echo "  build         - Build both backend and frontend"
	@echo "  test          - Run tests for backend and frontend"
	@echo "  lint          - Run linting for backend and frontend"
	@echo "  docker-up     - Build and start containers in background"
	@echo "  docker-down   - Stop and remove containers"
	@echo "  docker-up-e2e - Build and start containers with E2E test assets"
	@echo "  docker-down-e2e - Stop E2E containers"
	@echo "  test-e2e      - Run E2E tests using Playwright (requires docker-up-e2e)"
	@echo "  add-family    - Add a new family (requires docker-up)"
	@echo "                  Usage: make add-family NAME=\"The Smiths\""
	@echo "  add-user      - Add a new user (requires docker-up)"
	@echo "                  Usage: make add-user FAMILY=\"The Smiths\" UNAME=\"alice\" PASS=\"secret\""

# Build targets
build: build-backend build-frontend

build-backend:
	cd backend && go build -o ../bin/server cmd/server/main.go
	cd backend && go build -o ../bin/admin cmd/admin/main.go

build-frontend:
	cd frontend && npm install && npm run build

# Test targets
test: test-backend test-frontend test-e2e

test-backend:
	cd backend && go test ./...

test-frontend:
	cd frontend && npm test -- --passWithNoTests

test-e2e:
	@echo "Checking if Docker containers are running with E2E config..."
	@STARTED_CONTAINERS=false; \
	if ! docker compose -f docker-compose.yml -f docker-compose.e2e.yml ps nginx 2>/dev/null | grep -q "Up"; then \
		echo "🚀 Starting Docker containers with E2E config..."; \
		docker-compose -f docker-compose.yml -f docker-compose.e2e.yml up --build -d; \
		echo "⏳ Waiting for containers to be healthy..."; \
		sleep 5; \
		STARTED_CONTAINERS=true; \
	fi; \
	echo "✅ Docker containers are running. Starting E2E tests..."; \
	cd e2e && npx playwright test; \
	TEST_EXIT_CODE=$$?; \
	if [ "$$STARTED_CONTAINERS" = "true" ]; then \
		echo "🧹 Stopping Docker containers..."; \
		docker-compose -f docker-compose.yml -f docker-compose.e2e.yml down; \
	fi; \
	exit $$TEST_EXIT_CODE

# Lint targets
lint: lint-backend lint-frontend

lint-backend:
	cd backend && go tool golangci-lint run ./...


lint-frontend:
	cd frontend && npm run lint || echo "Linting failed or not configured"

# Docker targets
docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down

# E2E Docker targets (includes test assets)
docker-up-e2e:
	docker-compose -f docker-compose.yml -f docker-compose.e2e.yml up --build -d

docker-down-e2e:
	docker-compose -f docker-compose.yml -f docker-compose.e2e.yml down

# Administrative CLI targets
seed-test-data:
	@echo "Seeding test family and users..."
	docker-compose run --rm backend ./kincart-admin add-family --name "TestFamily"
	docker-compose run --rm backend ./kincart-admin add-user --family "TestFamily" --username "manager_test" --password "pass1234"
	docker-compose run --rm backend ./kincart-admin add-user --family "TestFamily" --username "shopper_test" --password "pass1234"
	docker-compose run --rm backend ./kincart-admin seed-categories "TestFamily"

add-family:
	@if [ -z "$(NAME)" ]; then echo "NAME is required. Example: make add-family NAME=\"Smiths\""; exit 1; fi
	docker-compose run --rm backend ./kincart-admin add-family --name "$(NAME)"

add-user:
	@if [ -z "$(FAMILY)" ] || [ -z "$(UNAME)" ] || [ -z "$(PASS)" ]; then \
		echo "FAMILY, UNAME, and PASS are required."; \
		echo "Example: make add-user FAMILY=\"Smiths\" UNAME=\"alice\" PASS=\"secret\""; \
		exit 1; \
	fi
	docker-compose run --rm backend ./kincart-admin add-user --family "$(FAMILY)" --username "$(UNAME)" --password "$(PASS)"
