ifneq (,$(wildcard .env))
  include .env
  export
endif

CMD_DIR := cmd/server
MIGRATIONS_PATH = migrations

clean: ## Remove build artifacts and cache
	@echo "🧹 Cleaning up..."
	@rm -rf bin/ *.out *.exe *.test
	go clean

run: ## Run the app
	@echo "🚀 Running app:"
	go run $(CMD_DIR)/main.go

tidy: ## Tidy go.mod and go.sum
	@echo "🧹 Tidying go.mod and go.sum..."
	go mod tidy

docker-up: ## Start docker containers
	docker-compose up -d

docker-down: ## Stop docker containers
	docker-compose down -v

migrate-up: ## Apply all up migrations
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

migrate-down: ## Apply all down migrations
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down

migrate-force: ## Force migration version
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" force $(version)

fix-dirty: ## Fix dirty migration state
	@echo "Checking for dirty migration state..."
	@MIGRATION_OUTPUT=$$(migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" version 2>&1 || true); \
	echo "$$MIGRATION_OUTPUT"; \
	if echo "$$MIGRATION_OUTPUT" | grep -q "dirty"; then \
		VERSION=$$(echo "$$MIGRATION_OUTPUT" | grep -oE '^[0-9]+'); \
		PREV_VERSION=$$((VERSION - 1)); \
		echo "Dirty state detected. Forcing to version $$PREV_VERSION"; \
		migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" force $$PREV_VERSION; \
		make migrate-down; \
	else \
		echo "Migration state is clean. No action needed."; \
	fi

migrate-retry: fix-dirty migrate-up ## Retry migrations after fixing dirty state

start-app: ## Start full stack (if docker available) and run app
	-docker-compose up -d
	@echo "⏳ Waiting 5s for DB to be ready..."
	@sleep 5
	make migrate-retry
	make run

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*?## "}; /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

test: ## Run all tests
	go test ./... 

test-force: ## Run tests without caching
	go test -count=1 ./... 

test-ci: ## Run tests with both race detection and coverage (used in CI)
	go test -race -coverprofile=coverage.out ./... 	
	go tool cover -func=coverage.out

test-log: ## Run all tests in the project, including showing logs
	go test -v ./... 

.PHONY: test test-force test-ci run tidy help clean test-log docker-up docker-down migrate-up migrate-down migrate-force fix-dirty migrate-retry start-app 