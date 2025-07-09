.PHONY: db-up db-down db-logs test build run clean

# Database commands
db-up:
	docker-compose up -d db

db-down:
	docker-compose down

db-logs:
	docker-compose logs -f db

# Development commands
test:
	go test -v ./...

build:
	go build -o bin/rss-comb app/main.go

run: db-up
	@if [ -f .env ]; then set -a && . ./.env && set +a; fi && go run app/main.go

clean:
	rm -rf bin/
	docker-compose down -v
