.PHONY: help run build test test-unit lint vet fmt proto \
        migrate-up migrate-down seed docker-build docker-run down clean

PROJECT_ENV ?= local

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Run / build ────────────────────────────────────────────────────────────────
run: ## Run reservation-service against local infra (postgres, redis, rabbitmq)
	PROJECT_ENV=$(PROJECT_ENV) go run ./cmd/reservation

build: ## Compile binary into bin/reservation
	CGO_ENABLED=0 go build -o bin/reservation ./cmd/reservation

# ── Tests ──────────────────────────────────────────────────────────────────────
test: ## Run all tests
	go test ./...

test-unit: ## Unit tests only (no infra)
	go test -short -race -count=1 ./pkg/... ./internal/...

# ── Quality ────────────────────────────────────────────────────────────────────
fmt: ## Format Go code
	gofmt -s -w .

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint (requires install)
	golangci-lint run ./...

# ── Code generation ────────────────────────────────────────────────────────────
proto: ## Regenerate api/proto/reservation/v1/{reservation.pb.go, reservation_grpc.pb.go}
	@which protoc >/dev/null || (echo "protoc not installed (brew install protobuf)" && exit 1)
	@which protoc-gen-go >/dev/null || (echo "protoc-gen-go not installed (go install google.golang.org/protobuf/cmd/protoc-gen-go@latest)" && exit 1)
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       api/proto/reservation/v1/reservation.proto

# ── DB migrations ──────────────────────────────────────────────────────────────
migrate-up: ## Apply DB migrations
	./scripts/migrate.sh up

migrate-down: ## Roll back one migration
	./scripts/migrate.sh down 1

seed: ## Seed 250 spots into reservation_service DB
	psql "postgres://postgres:postgres@localhost:5432/reservation_service?sslmode=disable" -f data/seed.sql

# ── Container / docker compose ────────────────────────────────────────────────
docker-build: ## Build the service container image
	docker build -t reservation-service:local .

docker-run: ## Bring up the service via deployments/docker-compose.yml (expects ../infra up)
	docker compose -f deployments/docker-compose.yml up --build

down: ## Tear down the service's docker compose stack
	docker compose -f deployments/docker-compose.yml down

# ── Housekeeping ───────────────────────────────────────────────────────────────
clean: ## Remove build artefacts
	rm -rf bin/ coverage.out
