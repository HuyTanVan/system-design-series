.PHONY: dev build up down restart logs clean test

# Run locally without Docker
dev:
	go run cmd/server/main.go

# Build the binary
build:
	go build -o bin/api cmd/api/main.go

# Start all containers
up:
	docker compose up --build -d

# Stop all containers
down:
	docker compose down

# Restart app container only
restart:
	docker compose restart app

# View logs
logs:
	docker compose logs -f app

# Remove containers + volumes (wipe DB)
clean:
	docker compose down -v

# Run tests
test:
	go test ./...

# Tidy dependencies
tidy:
	go mod tidy