init:
	@echo "Initializing project dependencies..."
	@go mod tidy
	@go mod download

format:
	@echo "Formatting code..."
	@go tool golangci-lint fmt

lint:
	@echo "Linting..."
	@go tool golangci-lint run ./...

test:
	@echo "Running tests..."
	@CGO_ENABLED=1 go tool gotestsum -- --race --vet= --count=2 -p=4 -tags=test ./...

WORKERS_NUM ?= 2
service-start:
	@cd docker && docker compose up -d --build --scale worker-service=$(WORKERS_NUM)

service-stop:
	@cd docker && docker compose stop



