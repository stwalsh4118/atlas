# Atlas API Makefile
# Database migration helpers using golang-migrate

# Database connection settings (override with environment variables)
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= postgres
DB_PASSWORD ?= postgres
DB_NAME ?= atlas
DB_SSL_MODE ?= disable

# Build the database URL  
DATABASE_URL := postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)

# Migration settings
MIGRATIONS_DIR := migrations
MIGRATE := migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)"

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: migrate-create
migrate-create: ## Create a new migration file (usage: make migrate-create NAME=add_users_table)
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=your_migration_name"; \
		exit 1; \
	fi
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME)

.PHONY: migrate-up
migrate-up: ## Run all pending migrations
	$(MIGRATE) up

.PHONY: migrate-down
migrate-down: ## Rollback the last migration
	$(MIGRATE) down 1

.PHONY: migrate-down-all
migrate-down-all: ## Rollback all migrations (WARNING: destructive)
	@echo "WARNING: This will rollback ALL migrations. Press Ctrl+C to cancel, or Enter to continue..."
	@read confirm
	$(MIGRATE) down -all

.PHONY: migrate-force
migrate-force: ## Force set migration version (usage: make migrate-force VERSION=1)
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make migrate-force VERSION=1"; \
		exit 1; \
	fi
	$(MIGRATE) force $(VERSION)

.PHONY: migrate-version
migrate-version: ## Show current migration version
	$(MIGRATE) version

.PHONY: migrate-goto
migrate-goto: ## Migrate to specific version (usage: make migrate-goto VERSION=2)
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make migrate-goto VERSION=2"; \
		exit 1; \
	fi
	$(MIGRATE) goto $(VERSION)

.PHONY: migrate-drop
migrate-drop: ## Drop everything in database (WARNING: extremely destructive)
	@echo "WARNING: This will drop EVERYTHING in the database. Type 'yes' to confirm:"
	@read confirm && [ "$$confirm" = "yes" ]
	$(MIGRATE) drop -f

.PHONY: db-url
db-url: ## Print the database URL (for debugging)
	@echo $(DATABASE_URL)

.PHONY: test
test: ## Run Go tests
	go test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: build
build: ## Build the API server
	go build -o bin/server ./cmd/server

.PHONY: run
run: ## Run the API server
	go run ./cmd/server/main.go

.PHONY: tidy
tidy: ## Tidy Go modules
	go mod tidy

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: fmt
fmt: ## Format Go code
	go fmt ./...

