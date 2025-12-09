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


DB_INTERNAL_URL := $(subst localhost:5433,postgres:5432,$(strip $(DATABASE_URL)))
MINIO_HOST_INT := minio:9000
MINIO_URL_INT := http://$(MINIO_HOST_INT)

docker-up: ## Start docker containers
	docker-compose up -d

docker-down: ## Stop docker containers
	docker-compose down -v

migrate-up: ## Apply all up migrations
	docker run --rm -v $(PWD)/$(MIGRATIONS_PATH):/migrations --network docai_default migrate/migrate \
		-path=/migrations/ -database "$(DB_INTERNAL_URL)" up

migrate-down: ## Apply all down migrations
	docker run --rm -v $(PWD)/$(MIGRATIONS_PATH):/migrations --network docai_default migrate/migrate \
		-path=/migrations/ -database "$(DB_INTERNAL_URL)" down

migrate-force: ## Force migration version
	docker run --rm -v $(PWD)/$(MIGRATIONS_PATH):/migrations --network docai_default migrate/migrate \
		-path=/migrations/ -database "$(DB_INTERNAL_URL)" force $(version)

fix-dirty: ## Fix dirty migration state
	@echo "Checking for dirty migration state..."
	@MIGRATION_OUTPUT=$$(docker run --rm -v $(PWD)/$(MIGRATIONS_PATH):/migrations --network docai_default migrate/migrate \
		-path=/migrations/ -database "$(DB_INTERNAL_URL)" version 2>&1 || true); \
	echo "$$MIGRATION_OUTPUT"; \
	if echo "$$MIGRATION_OUTPUT" | grep -q "dirty"; then \
		VERSION=$$(echo "$$MIGRATION_OUTPUT" | grep -oE '^[0-9]+'); \
		PREV_VERSION=$$((VERSION - 1)); \
		echo "Dirty state detected. Forcing to version $$PREV_VERSION"; \
		make migrate-force version=$$PREV_VERSION; \
		make migrate-down; \
	else \
		echo "Migration state is clean. No action needed."; \
	fi

migrate-retry: fix-dirty migrate-up ## Retry migrations after fixing dirty state

minio-setup: ## Configure MinIO bucket and policy
	@echo "Setting up MinIO..."
	@docker run --rm --network docai_default --entrypoint /bin/sh minio/mc -c "\
	until mc alias set myminio $(MINIO_URL_INT) $(strip $(MINIO_ACCESS_KEY)) $(strip $(MINIO_SECRET_KEY)); do echo 'Waiting for MinIO...'; sleep 1; done; \
	mc mb --ignore-existing myminio/$(strip $(MINIO_BUCKET)); \
	mc anonymous set public myminio/$(strip $(MINIO_BUCKET));"

start-app: docker-up minio-setup migrate-retry run ## Start full stack and run app

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

.PHONY: test test-force test-ci run tidy help clean test-log docker-up docker-down migrate-up migrate-down migrate-force fix-dirty migrate-retry minio-setup start-app 