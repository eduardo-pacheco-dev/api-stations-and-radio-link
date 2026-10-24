.PHONY: run build fmt vet test test-race test-integration cover swagger up down logs clean

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

test-integration:
	go test -tags=integration ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/api/main.go -o docs --parseInternal

up:
	docker compose up -d mysql

down:
	docker compose down

logs:
	docker compose logs -f mysql

clean:
	rm -rf bin coverage.out coverage.html
