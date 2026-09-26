#!/usr/bin/env bash

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

services=(
    "business-service"
    "review-service"
    "search-service"
)

for service in "${services[@]}"; do
    echo "Starting $service..."

    (
        cd "$ROOT/$service" || exit 1
        exec go run cmd/api/main.go
    ) &
done

wait