APP_NAME := go-user-system
IMAGE_NAME := go-user-system:dev

.PHONY: run test build clean tidy docker-build compose-up compose-down compose-logs

run:
	go run ./cmd

test:
	go test ./...

build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -o bin/$(APP_NAME) ./cmd

clean:
	rm -rf bin

tidy:
	go mod tidy

docker-build:
	docker build -t $(IMAGE_NAME) .

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down

compose-logs:
	docker compose logs -f app
