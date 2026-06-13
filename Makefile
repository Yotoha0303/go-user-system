APP_NAME := go-user-system
IMAGE_NAME := go-user-system:dev

GOPATH := $(shell go env GOPATH)
GOOSE ?= $(subst \,/,$(GOPATH))/bin/goose.exe
GOOSE_ENV ?= .env.goose

.DEFAULT_GOAL := help

.PHONY: help run test coverage coverage-html integration-test race-test vet build build-windows build-linux clean tidy \
	goose-version migrate-create migrate-validate migrate-status migrate-version migrate-up migrate-up-by-one migrate-down migrate-redo migrate-reset migrate-fix \
	docker-build compose-up compose-down compose-logs ci

help:
	@echo Usage: make target
	@echo App:
	@echo   run                 Run the application locally
	@echo   build               Build local binary
	@echo   build-windows       Build Windows binary
	@echo   build-linux         Build Linux binary
	@echo   clean               Remove build artifacts
	@echo   tidy                Run go mod tidy
	@echo Quality:
	@echo   test                Run all tests
	@echo   integration-test    Run integration tests
	@echo   coverage            Run tests and print coverage summary
	@echo   coverage-html       Generate HTML coverage report
	@echo   race-test           Run tests with race detector
	@echo   vet                 Run go vet
	@echo   ci                  Run test, vet, build and docker-build
	@echo Migration:
	@echo   goose-version       Print goose version
	@echo   migrate-create      Create migration. Usage: make migrate-create name=create_users
	@echo   migrate-validate    Validate migration files
	@echo   migrate-status      Show migration status
	@echo   migrate-version     Show current database migration version
	@echo   migrate-up          Apply all pending migrations
	@echo   migrate-up-by-one   Apply one pending migration
	@echo   migrate-down        Roll back one migration
	@echo   migrate-redo        Re-run latest migration
	@echo   migrate-reset       Roll back all migrations
	@echo   migrate-fix         Convert timestamps to sequential ordering
	@echo Docker:
	@echo   docker-build        Build Docker image
	@echo   compose-up          Start Docker Compose stack
	@echo   compose-down        Stop Docker Compose stack
	@echo   compose-logs        Follow app logs

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
	go test ./... -run Integration -v

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

goose-version:
	"$(GOOSE)" -version

migrate-create:
	$(if $(name),,$(error name is required. Usage: make migrate-create name=create_users))
	"$(GOOSE)" -env "$(GOOSE_ENV)" -s create "$(name)" sql

migrate-validate:
	"$(GOOSE)" -env "$(GOOSE_ENV)" validate

migrate-status:
	"$(GOOSE)" -env "$(GOOSE_ENV)" status

migrate-version:
	"$(GOOSE)" -env "$(GOOSE_ENV)" version

migrate-up:
	"$(GOOSE)" -env "$(GOOSE_ENV)" up

migrate-up-by-one:
	"$(GOOSE)" -env "$(GOOSE_ENV)" up-by-one

migrate-down:
	"$(GOOSE)" -env "$(GOOSE_ENV)" down

migrate-redo:
	"$(GOOSE)" -env "$(GOOSE_ENV)" redo

migrate-reset:
	"$(GOOSE)" -env "$(GOOSE_ENV)" reset

migrate-fix:
	"$(GOOSE)" -env "$(GOOSE_ENV)" fix

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
