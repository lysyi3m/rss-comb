.PHONY: db-up db-down db-logs test build run clean stop-all cleanup-all

# Database commands
db-up:
	docker-compose -p rss-comb-dev up -d db

db-down:
	docker-compose -p rss-comb-dev down

db-logs:
	docker-compose -p rss-comb-dev logs -f db

# Development commands
test:
	go test -v ./...

build:
	@mkdir -p bin
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	go build -ldflags "-X github.com/lysyi3m/rss-comb/app/version.Version=$$VERSION" -o bin/rss-comb app/main.go

run: db-up
	@if [ -f .env ]; then set -a && . ./.env && set +a; fi; \
	VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	go run -ldflags "-X github.com/lysyi3m/rss-comb/app/version.Version=$$VERSION" app/main.go

clean:
	rm -rf bin/
	docker-compose -p rss-comb-dev down -v

# Stop development RSS Comb processes (not production containers)
stop-all:
	@echo "Stopping development RSS Comb processes..."
	@-pkill -f "go run.*main.go" 2>/dev/null
	@-pkill -f "/home/.*/.cache/go-build.*/main" 2>/dev/null
	@-pkill -f "bin/rss-comb" 2>/dev/null
	@echo "Development RSS Comb processes stopped"

# Complete cleanup: stop processes, remove containers, clean build artifacts
cleanup-all: stop-all
	@echo "Performing complete cleanup..."
	docker-compose -p rss-comb-dev down -v --remove-orphans
	rm -rf bin/
	go clean -cache
	go clean -modcache || true
	@echo "Complete cleanup finished - all processes stopped, containers removed, caches cleared"
