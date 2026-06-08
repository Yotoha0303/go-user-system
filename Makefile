APP_NAME := go-user-system
IMAGE_NAME := go-user-system:dev

.DEFAULT_GOAL := help

.PHONY: help run test coverage coverage-html integration-test race-test vet build build-windows build-linux clean tidy docker-build compose-up compose-down compose-logs ci

help:
	@echo Usage: make target
	@echo Targets:
	@echo   help              Show this help message
	@echo   run               Run the application locally
	@echo   test              Run all tests
	@echo   coverage          Run tests and print coverage summary
	@echo   coverage-html     Generate HTML coverage report
	@echo   integration-test  Run integration tests
	@echo   race-test         Run tests with race detector
	@echo   vet               Run go vet
	@echo   build             Build local binary
	@echo   build-windows     Build Windows binary
	@echo   build-linux       Build Linux binary
	@echo   clean             Remove build artifacts
	@echo   tidy              Run go mod tidy
	@echo   docker-build      Build Docker image
	@echo   compose-up        Start Docker Compose stack
	@echo   compose-down      Stop Docker Compose stack
	@echo   compose-logs      Follow app logs
	@echo   ci                Run test, vet, build and docker-build

run:
	go run ./cmd

test:
	go test ./...

coverage:
	go test -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

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
	rm -rf bin coverage.out coverage.html

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

ci:
	$(MAKE) test
	$(MAKE) vet
	$(MAKE) build
	$(MAKE) docker-build
