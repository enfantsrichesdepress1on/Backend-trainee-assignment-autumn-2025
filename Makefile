BINARY_NAME=app
MAIN_PKG=./cmd/app

DOCKER_IMAGE=avito-pr-service

.PHONY: build test run docker-build docker-up docker-down

build:
	go build -o bin/$(BINARY_NAME) $(MAIN_PKG)

test:
	go test ./...

run:
	DATABASE_URL=postgres://pr_user:pr_password@localhost:5432/pr_service?sslmode=disable \
	HTTP_ADDR=:8080 \
	go run $(MAIN_PKG)

docker-build:
	docker build -t $(DOCKER_IMAGE) .

docker-up:
	docker-compose up --build

docker-down:
	docker-compose down
