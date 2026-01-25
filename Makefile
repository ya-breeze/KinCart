.PHONY: build test lint docker-up docker-down add-family add-user help

# Default target
help:
	@echo "KinCart Makefile Tasks:"
	@echo "  build         - Build both backend and frontend"
	@echo "  test          - Run tests for backend and frontend"
	@echo "  lint          - Run linting for backend and frontend"
	@echo "  docker-up     - Build and start containers in background"
	@echo "  docker-down   - Stop and remove containers"
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
test: test-backend test-frontend

test-backend:
	cd backend && go test ./...

test-frontend:
	cd frontend && npm test -- --passWithNoTests

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
