.PHONY: dev-up dev-down test build run migrate docker-build docker-up docker-down deploy clean

# Development commands
dev-up:
	docker-compose up -d db redis

dev-down:
	docker-compose down

test:
	go test -v ./...

build:
	go build -o bin/rss-comb cmd/server/main.go

run: dev-up
	go run cmd/server/main.go

migrate:
	$(shell go env GOPATH)/bin/migrate -path migrations -database "postgres://rss_user:rss_password@localhost:5432/rss_bridge?sslmode=disable" up

# Docker commands
docker-build:
	./scripts/build.sh

docker-up:
	docker-compose -f docker-compose.prod.yml up -d

docker-down:
	docker-compose -f docker-compose.prod.yml down

docker-logs:
	docker-compose -f docker-compose.prod.yml logs -f

# Deployment
deploy:
	./scripts/deploy.sh

# Cleanup
clean:
	rm -rf bin/
	docker system prune -f
	docker volume prune -f