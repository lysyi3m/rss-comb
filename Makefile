.PHONY: dev-test dev-build dev-run dev-stop dev-clean

# Development commands
dev-test:
	go test -v ./...

dev-build:
	@mkdir -p bin
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	go build -ldflags "-X github.com/lysyi3m/rss-comb/app/cfg.Version=$$VERSION" -o bin/rss-comb app/main.go

# Runs the built binary rather than `go run`, so dev-stop can match this project's
# process by its absolute path.
dev-run: dev-build
	@echo "Starting RSS Comb with development database..."
	@mkdir -p data media
	@DB_PATH=./data/rss-comb-dev.db \
	MEDIA_DIR=./media \
	DEV_UID=$$(id -u) DEV_GID=$$(id -g) \
	YT_DLP_CMD="docker compose -p rss-comb-dev run --rm yt-dlp" \
	"$(CURDIR)/bin/rss-comb"

# Stop development RSS Comb processes (not production containers)
dev-stop:
	@echo "Stopping development RSS Comb processes..."
	@-pkill -f "^$(CURDIR)/bin/rss-comb" 2>/dev/null
	@# Scope teardown to the dev compose project. Do NOT filter by `ancestor=` on the
	@# app image: dev yt-dlp now runs the production image, so an ancestor filter also
	@# matches a running production container and would stop it.
	@-docker compose -p rss-comb-dev down --remove-orphans 2>/dev/null
	@echo "Development RSS Comb processes stopped"

# Complete development cleanup: stop processes, clean build artifacts and dev database
dev-clean: dev-stop
	@echo "Performing complete development cleanup..."
	rm -f data/rss-comb-dev.db data/rss-comb-dev.db-wal data/rss-comb-dev.db-shm
	rm -rf bin/
	go clean -cache
	go clean -modcache || true
	@echo "Complete development cleanup finished"
