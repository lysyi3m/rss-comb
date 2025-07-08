.PHONY: dev-up dev-down test build run docker-build docker-up docker-down deploy clean

# Development commands
dev-up:
	docker-compose up -d db

dev-down:
	docker-compose down

test:
	go test -v ./...

build:
	go build -o bin/rss-comb app/main.go

run: dev-up
	go run app/main.go


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