.PHONY: db-up db-down db-logs test build run clean

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
