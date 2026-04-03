.PHONY: dev-test dev-build dev-run dev-stop dev-clean

# Development commands
dev-test:
	go test -v ./...

dev-build:
	@mkdir -p bin
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	go build -ldflags "-X github.com/lysyi3m/rss-comb/app/cfg.Version=$$VERSION" -o bin/rss-comb app/main.go

dev-run:
	@echo "Starting RSS Comb with development database..."
	@mkdir -p data media
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	DB_PATH=./data/rss-comb-dev.db \
	MEDIA_DIR=./media \
	YT_DLP_CMD="docker compose -p rss-comb-dev run --rm yt-dlp" \
	go run -ldflags "-X github.com/lysyi3m/rss-comb/app/cfg.Version=$$VERSION" app/main.go

# Stop development RSS Comb processes (not production containers)
dev-stop:
	@echo "Stopping development RSS Comb processes..."
	@-pkill -f "go run.*main.go" 2>/dev/null
	@-pkill -f "/home/.*/.cache/go-build.*/main" 2>/dev/null
	@-pkill -f "bin/rss-comb" 2>/dev/null
	@-docker ps -q --filter "ancestor=jauderho/yt-dlp:2026.03.17" | xargs -r docker stop 2>/dev/null
	@echo "Development RSS Comb processes stopped"

# Complete development cleanup: stop processes, clean build artifacts and dev database
dev-clean: dev-stop
	@echo "Performing complete development cleanup..."
	rm -f data/rss-comb-dev.db data/rss-comb-dev.db-wal data/rss-comb-dev.db-shm
	rm -rf bin/
	go clean -cache
	go clean -modcache || true
	@echo "Complete development cleanup finished"
