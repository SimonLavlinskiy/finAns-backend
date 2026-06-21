.PHONY: up down run migrate migrate-down seed sqlc mocks test test-cover test-integration lint swagger build tidy

DATABASE_URL ?= postgres://finans:finans@localhost:5435/finans?sslmode=disable

up:
	docker compose up -d postgres

down:
	docker compose down

run:
	go run ./cmd/app

migrate:
	DATABASE_URL=$(DATABASE_URL) go run ./cmd/migrate -direction up

migrate-down:
	DATABASE_URL=$(DATABASE_URL) go run ./cmd/migrate -direction down

seed:
	psql $(DATABASE_URL) -f testdata/fixtures/tags.sql
	psql $(DATABASE_URL) -f testdata/fixtures/transactions.sql

sqlc:
	sqlc generate

mocks:
	mockery

test:
	go test ./... -count=1

test-cover:
	go test ./... -count=1 -coverprofile=coverage.out
	go tool cover -func=coverage.out

test-integration:
	go test ./... -count=1 -tags=integration

lint:
	golangci-lint run ./...

swagger:
	swag init -g cmd/app/main.go -o docs --parseDependency --parseInternal

build:
	go build -o bin/finans-api ./cmd/app

tidy:
	go mod tidy
