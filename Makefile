.PHONY: run build fmt vet test test-race cover swagger migrate seed up down logs clean

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

fmt:
	gofmt -w cmd internal docs

vet:
	go vet ./...

test:
	go test ./...

test-race:
	go test -race ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/api/main.go -o docs --parseInternal

migrate:
	go run ./cmd/migrate

seed:
	go run ./cmd/seed

up:
	docker compose up -d mysql

down:
	docker compose down

logs:
	docker compose logs -f mysql

clean:
	rm -rf bin coverage.out coverage.html
