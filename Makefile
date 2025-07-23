.PHONY: dev-db-up dev-db-down dev-db-logs dev-test dev-build dev-run dev-stop dev-clean

# Development database commands
dev-db-up:
	docker-compose -p rss-comb-dev up -d db

dev-db-down:
	docker-compose -p rss-comb-dev down

dev-db-logs:
	docker-compose -p rss-comb-dev logs -f db

# Development commands
dev-test:
	go test -v ./...

dev-build:
	@mkdir -p bin
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	go build -ldflags "-X github.com/lysyi3m/rss-comb/app/cfg.Version=$$VERSION" -o bin/rss-comb app/main.go

dev-run: dev-db-up
	@echo "Starting RSS Comb with development database..."
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	DB_HOST=localhost \
	DB_PORT=5432 \
	DB_USER=rss_comb_dev_user \
	DB_PASSWORD=rss_comb_dev_password \
	DB_NAME=rss_comb_dev \
	go run -ldflags "-X github.com/lysyi3m/rss-comb/app/cfg.Version=$$VERSION" app/main.go


# Stop development RSS Comb processes (not production containers)
dev-stop:
	@echo "Stopping development RSS Comb processes..."
	@-pkill -f "go run.*main.go" 2>/dev/null
	@-pkill -f "/home/.*/.cache/go-build.*/main" 2>/dev/null
	@-pkill -f "bin/rss-comb" 2>/dev/null
	@echo "Development RSS Comb processes stopped"

# Complete development cleanup: stop processes, remove containers, clean build artifacts
dev-clean: dev-stop
	@echo "Performing complete development cleanup..."
	docker-compose -p rss-comb-dev down -v --remove-orphans
	rm -rf bin/
	go clean -cache
	go clean -modcache || true
	@echo "Complete development cleanup finished - all processes stopped, containers removed, caches cleared"
