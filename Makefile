APP_NAME := go-user-system
IMAGE_NAME := go-user-system:dev

.PHONY: run test coverage coverage-html integration-test race-test vet build build-windows build-linux clean tidy docker-build compose-up compose-down compose-logs ci

run:
	go run ./cmd

test:
	go test ./...

coverage:
	go test ./... "-coverprofile=coverage.out" -covermode=atomic
	go tool cover "-func=coverage.out"

coverage-html: coverage
	go tool cover "-html=coverage.out" -o coverage.html

integration-test:
	go test ./... -run Integration

race-test:
	go test -race ./...

vet:
	go vet ./...

build:
	go build -o bin/$(APP_NAME) ./cmd

build-windows: export GOOS=windows
build-windows:
	go build -o bin/$(APP_NAME).exe ./cmd

build-linux: export CGO_ENABLED=0
build-linux: export GOOS=linux
build-linux:
	go build -o bin/$(APP_NAME) ./cmd

clean:
	rm -rfv bin/*

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

ci: test vet build docker-build
