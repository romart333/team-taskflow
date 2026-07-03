APP_DIR := app
BUILD_DIR := build

.PHONY: build run test test-integration lint up down tidy

build:
	cd $(APP_DIR) && go build -o ../$(BUILD_DIR)/server ./cmd/server

run:
	cd $(APP_DIR) && go run ./cmd/server

test: lint
	cd $(APP_DIR) && go test ./...

test-integration:
	cd $(APP_DIR) && go test -tags integration -count=1 ./test/...

lint:
	cd $(APP_DIR) && go vet ./... && golangci-lint run ./...

up:
	docker compose up -d mysql redis

down:
	docker compose down

tidy:
	cd $(APP_DIR) && go mod tidy
