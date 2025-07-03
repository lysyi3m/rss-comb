.PHONY: dev-up dev-down test build run migrate

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