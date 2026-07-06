APP_DIR := app
BUILD_DIR := build

.PHONY: build run test test-unit test-integration lint up down tidy generate

build:
	cd $(APP_DIR) && go build -o ../$(BUILD_DIR)/server ./cmd/server

run: $(APP_DIR)/configs/config.yaml
	cd $(APP_DIR) && go run ./cmd/server

$(APP_DIR)/configs/config.yaml:
	cp $(APP_DIR)/configs/config.yaml.example $(APP_DIR)/configs/config.yaml

test: lint test-unit

test-unit:
	cd $(APP_DIR) && go test -race ./...

test-integration:
	cd $(APP_DIR) && go test -tags integration -count=1 ./test/...

lint:
	cd $(APP_DIR) && golangci-lint run ./...

up:
	docker compose up -d mysql redis

down:
	docker compose down

tidy:
	cd $(APP_DIR) && go mod tidy

generate:
	cd $(APP_DIR) && go tool mockery
