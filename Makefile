.PHONY: run build test lint migrate-up migrate-down docker-build docker-run down

PROJECT_ENV ?= local

run:
	go run ./cmd/reservation

build:
	CGO_ENABLED=0 go build -o bin/reservation ./cmd/reservation

test:
	go test ./...

lint:
	golangci-lint run ./...

migrate-up:
	./scripts/migrate.sh up

migrate-down:
	./scripts/migrate.sh down 1

docker-build:
	docker build -t reservation-service:local .

docker-run:
	docker compose -f deployments/docker-compose.yml up --build

down:
	docker compose -f deployments/docker-compose.yml down
