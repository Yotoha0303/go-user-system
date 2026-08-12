APP_NAME := go-user-system
IMAGE_NAME := go-user-system:dev
VERSION ?= dev
COMMIT ?= unknown
BUILD_TIME ?= unknown
PROMETHEUS_IMAGE := quay.io/prometheus/prometheus:v3.5.5
OBSERVABILITY_COMPOSE := docker compose -f compose.yaml -f compose.observability.yaml
ifeq ($(OS),Windows_NT)
POWERSHELL ?= powershell
else
POWERSHELL ?= pwsh
endif
GO_LDFLAGS := -X go-user-system/internal/buildinfo.Version=$(VERSION) -X go-user-system/internal/buildinfo.Commit=$(COMMIT) -X go-user-system/internal/buildinfo.BuildTime=$(BUILD_TIME)

# Kubernetes
K8S_DIR := k8s
K8S_NAMESPACE := go-user-system
K8S_DEPLOYMENT := go-user-system
K8S_BACKEND_IMAGE ?= ghcr.io/yotoha0303/go-user-system-backend:v1.0.0-rc.3
K8S_FRONTEND_IMAGE ?= ghcr.io/yotoha0303/go-user-system-frontend:v1.0.0-rc.3
KIND_NAME := go-user-system

GOPATH := $(shell go env GOPATH)
GOOSE ?= $(subst \,/,$(GOPATH))/bin/goose.exe
GOOSE_ENV ?= .env.goose
GOLANGCI_LINT ?= golangci-lint
SWAGGER_CACHE ?= $(CURDIR)/.cache/swagger
CALLVIS_VERSION ?= v0.7.1
CALLVIS_CACHE ?= $(CURDIR)/.cache/go-callvis
CALLVIS_OUTPUT ?= docs/backend-callgraph
CALLVIS_HTTP ?= 127.0.0.1:7878

.DEFAULT_GOAL := help

.PHONY: help run test coverage coverage-html integration-test race-test vet lint lint-fix security frontend-check e2e bootstrap-admin swagger callvis callvis-serve \
	build build-windows build-linux clean tidy \
	goose-version migrate-create migrate-validate migrate-status migrate-version migrate-up migrate-up-by-one migrate-down migrate-redo migrate-reset migrate-fix \
	docker-build compose-up compose-down compose-logs observability-validate observability-up observability-down ops-backup ops-restore-drill ci \
	k8s-namespace k8s-build k8s-build-kind k8s-build-push k8s-port-forward k8s-all \
	k8s-deploy k8s-undeploy k8s-status k8s-logs k8s-apply k8s-dry-run k8s-restart k8s-validate k8s-wait \
	k8s-info \
	kind-create kind-up kind-down kind-load-deps

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
	@echo   security            Run backend and frontend dependency scans
	@echo   frontend-check      Run frontend lint, tests and production build
	@echo   e2e                 Run Playwright tests against the Compose stack
	@echo   bootstrap-admin     Create the first admin using BOOTSTRAP_ADMIN_* env vars
	@echo   swagger             Generate Swagger docs with swaggo
	@echo   callvis             Generate the backend RTA call graph
	@echo   callvis-serve       Start the interactive call graph viewer
	@echo   ci                  Run test, vet, build and docker-build
	@echo Kubernetes:
	@echo   k8s-namespace       Create/ensure namespace
	@echo   k8s-build           Build Docker image
	@echo   k8s-build-kind      Build image and load into kind cluster
	@echo   k8s-build-push      Build image and push to registry
	@echo   k8s-deploy          Deploy all resources to Kubernetes
	@echo   k8s-undeploy        Remove all resources from Kubernetes
	@echo   k8s-status          Show deployment status
	@echo   k8s-logs            Follow logs \(all containers\)
	@echo   k8s-apply           Deploy + wait
	@echo   k8s-all             Build + load + deploy + wait + status
	@echo   k8s-dry-run         Dry-run apply all resources
	@echo   k8s-restart         Restart deployment
	@echo   k8s-validate        Validate YAML \(dry-run\)
	@echo   k8s-wait            Wait for deployment to be ready \(up to 5min\)
	@echo   k8s-port-forward    Port-forward backend 8082 to localhost
	@echo Debug:
	@echo   k8s-info            Full cluster status and logs
	@echo Kind:
	@echo   kind-create         Create kind cluster
	@echo   kind-load-deps      Pre-load mysql, nginx images into kind
	@echo   kind-up             Create cluster + load deps + build + deploy + wait
	@echo   kind-down           Delete kind cluster \(and namespace\)
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
	@echo   observability-up    Start the full stack with Prometheus
	@echo   observability-down  Stop the full stack with Prometheus
	@echo   observability-validate Validate Prometheus and Compose configuration
	@echo   ops-backup          Back up Compose MySQL with checksum and manifest
	@echo   ops-restore-drill   Restore BACKUP_PATH into RESTORE_DATABASE \(must end with _restore_test\)
	@echo golangci:
	@echo   lint                Run golangci-lint
	@echo   lint-fix            Apply supported automatic lint fixes

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

security:
	go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
	cd frontend && npm audit --audit-level=high --registry=https://registry.npmjs.org

frontend-check:
	cd frontend && npm ci && npm run check

e2e:
	cd frontend && npm run test:e2e

bootstrap-admin:
	$(if $(BOOTSTRAP_ADMIN_USERNAME),,$(error BOOTSTRAP_ADMIN_USERNAME is required))
	$(if $(BOOTSTRAP_ADMIN_PASSWORD),,$(error BOOTSTRAP_ADMIN_PASSWORD is required))
	docker compose run --rm -e BOOTSTRAP_ADMIN_USERNAME -e BOOTSTRAP_ADMIN_PASSWORD app bootstrap-admin

swagger: export GOCACHE := $(SWAGGER_CACHE)
swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -d ./cmd,./internal/handler,./internal/request,./internal/response -g main.go -o docs --parseInternal

# Do not add -nostd: go-callvis misclassifies this dotless module path as stdlib.
callvis callvis-serve: export GOCACHE := $(CALLVIS_CACHE)
callvis:
	go run github.com/ofabry/go-callvis@$(CALLVIS_VERSION) -algo rta -focus= -group pkg,type -limit $(APP_NAME) -rankdir LR -file $(CALLVIS_OUTPUT) ./cmd

callvis-serve:
	go run github.com/ofabry/go-callvis@$(CALLVIS_VERSION) -algo rta -focus= -group pkg,type -limit $(APP_NAME) -rankdir LR -http $(CALLVIS_HTTP) -skipbrowser ./cmd

build:
	go build -trimpath -ldflags "$(GO_LDFLAGS)" -o bin/$(APP_NAME) ./cmd

build-windows: export GOOS=windows
build-windows:
	go build -trimpath -ldflags "$(GO_LDFLAGS)" -o bin/$(APP_NAME).exe ./cmd

build-linux: export CGO_ENABLED=0
build-linux: export GOOS=linux
build-linux:
	go build -trimpath -ldflags "$(GO_LDFLAGS)" -o bin/$(APP_NAME) ./cmd

clean:
	rm -rf bin coverage.out coverage.html "$(SWAGGER_CACHE)" "$(CALLVIS_CACHE)"

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

lint:
	$(GOLANGCI_LINT) run ./...

lint-fix:
	$(GOLANGCI_LINT) run --fix ./...

docker-build:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_TIME=$(BUILD_TIME) -t $(IMAGE_NAME) .

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down

compose-logs:
	docker compose logs -f app

observability-validate:
	$(OBSERVABILITY_COMPOSE) config --quiet
	docker run --rm --entrypoint /bin/promtool -v "$(CURDIR)/deploy/monitoring:/etc/prometheus:ro" $(PROMETHEUS_IMAGE) check config /etc/prometheus/prometheus.yml
	docker run --rm --entrypoint /bin/promtool -v "$(CURDIR)/deploy/monitoring:/etc/prometheus:ro" $(PROMETHEUS_IMAGE) check rules /etc/prometheus/rules/go-user-system.yml

observability-up:
ifeq ($(OS),Windows_NT)
	$(POWERSHELL) -NoProfile -Command "$$env:APP_VERSION='$(VERSION)'; $$env:APP_COMMIT='$(COMMIT)'; $$env:APP_BUILD_TIME='$(BUILD_TIME)'; docker compose -f compose.yaml -f compose.observability.yaml up -d --build --wait"
else
	APP_VERSION=$(VERSION) APP_COMMIT=$(COMMIT) APP_BUILD_TIME=$(BUILD_TIME) $(OBSERVABILITY_COMPOSE) up -d --build --wait
endif

observability-down:
	$(OBSERVABILITY_COMPOSE) down

ops-backup:
	$(POWERSHELL) -NoProfile -File scripts/ops/backup-mysql.ps1

ops-restore-drill:
	$(if $(BACKUP_PATH),,$(error BACKUP_PATH is required))
	$(POWERSHELL) -NoProfile -File scripts/ops/restore-mysql.ps1 -BackupPath "$(BACKUP_PATH)" -RestoreDatabase "$(or $(RESTORE_DATABASE),go_user_system_restore_test)" -ConfirmRestore

ci:
	$(MAKE) lint
	$(MAKE) test
	$(MAKE) race-test
	$(MAKE) vet
	$(MAKE) security
	$(MAKE) frontend-check
	$(MAKE) build
	$(MAKE) docker-build

# Kubernetes
k8s-namespace:
	kubectl create namespace $(K8S_NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -

k8s-build:
	docker build --platform linux/amd64 --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_TIME=$(BUILD_TIME) -t $(K8S_BACKEND_IMAGE) .
	docker build --platform linux/amd64 --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILD_TIME=$(BUILD_TIME) -t $(K8S_FRONTEND_IMAGE) ./frontend

k8s-build-kind: k8s-build
	kind load docker-image $(K8S_BACKEND_IMAGE) $(K8S_FRONTEND_IMAGE) --name $(KIND_NAME)

# Bypass "kind load" for multi-arch images. kind uses --all-platforms
# which fails when Docker only has linux/amd64 blobs.
kind-load-deps:
	docker pull --platform linux/amd64 mysql:8.4
	docker pull --platform linux/amd64 redis:7.4-alpine
	docker save mysql:8.4 | docker exec -i $(KIND_NAME)-control-plane ctr -n k8s.io images import --platform linux/amd64 --base-name docker.io/library/mysql:8.4 -
	docker save redis:7.4-alpine | docker exec -i $(KIND_NAME)-control-plane ctr -n k8s.io images import --platform linux/amd64 --base-name docker.io/library/redis:7.4-alpine -

k8s-build-push: k8s-build
	docker push $(K8S_BACKEND_IMAGE)
	docker push $(K8S_FRONTEND_IMAGE)

k8s-deploy: k8s-namespace
	kubectl get secret go-user-system-secret -n $(K8S_NAMESPACE)
	kubectl apply -f $(K8S_DIR)/configmap.yaml
	kubectl apply -f $(K8S_DIR)/mysql.yaml
	kubectl apply -f $(K8S_DIR)/redis.yaml
	kubectl wait --for=condition=available deployment/go-user-system-mysql -n $(K8S_NAMESPACE) --timeout=600s
	kubectl wait --for=condition=available deployment/go-user-system-redis -n $(K8S_NAMESPACE) --timeout=600s
	kubectl apply -f $(K8S_DIR)/migration-job.yaml
	kubectl wait --for=condition=complete -f $(K8S_DIR)/migration-job.yaml --timeout=600s
	kubectl apply -f $(K8S_DIR)/service.yaml
	kubectl apply -f $(K8S_DIR)/frontend-service.yaml
	kubectl apply -f $(K8S_DIR)/deployment.yaml
	kubectl apply -f $(K8S_DIR)/frontend-deployment.yaml
	kubectl apply -f $(K8S_DIR)/ingress.yaml

k8s-undeploy:
	kubectl delete namespace $(K8S_NAMESPACE) --ignore-not-found=true

k8s-status:
	@echo "=== Backend ==="
	kubectl get pods,svc,deployment -n $(K8S_NAMESPACE) -l app=$(K8S_DEPLOYMENT) 2>/dev/null || true
	@echo ""
	@echo "=== MySQL ==="
	kubectl get pods,svc,deployment,pvc -n $(K8S_NAMESPACE) -l app=$(K8S_DEPLOYMENT)-mysql 2>/dev/null || true
	@echo ""
	@echo "=== Redis ==="
	kubectl get pods,svc,deployment,pvc -n $(K8S_NAMESPACE) -l app=$(K8S_DEPLOYMENT)-redis 2>/dev/null || true
	@echo ""
	@echo "=== Frontend ==="
	kubectl get pods,svc,deployment -n $(K8S_NAMESPACE) -l app=$(K8S_DEPLOYMENT)-frontend 2>/dev/null || true
	@echo ""
	@echo "=== ConfigMaps & Secrets ==="
	kubectl get configmap,secret -n $(K8S_NAMESPACE) 2>/dev/null || true

k8s-logs:
	kubectl logs -f -n $(K8S_NAMESPACE) -l app=$(K8S_DEPLOYMENT) --all-containers

k8s-apply: k8s-deploy
	$(MAKE) k8s-wait

k8s-all: k8s-build-kind k8s-deploy k8s-wait
	@echo "=== Deployment Complete ==="
	$(MAKE) k8s-status

k8s-dry-run:
	kubectl apply -f $(K8S_DIR)/ --dry-run=server -o yaml

k8s-restart:
	kubectl rollout restart deployment/$(K8S_DEPLOYMENT) -n $(K8S_NAMESPACE)

k8s-validate:
	kubectl apply -f $(K8S_DIR)/ --validate --dry-run=server

k8s-wait:
	kubectl wait --for=condition=available deployment/$(K8S_DEPLOYMENT) -n $(K8S_NAMESPACE) --timeout=600s
	kubectl wait --for=condition=ready pod -l app=$(K8S_DEPLOYMENT)-mysql -n $(K8S_NAMESPACE) --timeout=600s 2>/dev/null || true
	kubectl wait --for=condition=ready pod -l app=$(K8S_DEPLOYMENT)-redis -n $(K8S_NAMESPACE) --timeout=600s

k8s-port-forward:
	kubectl port-forward -n $(K8S_NAMESPACE) svc/$(K8S_DEPLOYMENT) 8082:8082

# 调试与信息
k8s-info:
	@echo "=== Namespace: $(K8S_NAMESPACE) ==="
	kubectl get pods -n $(K8S_NAMESPACE) -l app=$(K8S_DEPLOYMENT) -o wide
	@echo ""
	kubectl get svc -n $(K8S_NAMESPACE) $(K8S_DEPLOYMENT)
	@echo ""
	kubectl get deployment -n $(K8S_NAMESPACE) $(K8S_DEPLOYMENT) -o yaml
	@echo ""
	kubectl get configmap -n $(K8S_NAMESPACE) $(K8S_DEPLOYMENT)-config -o yaml
	@echo "=== Recent Pod Logs ==="
	kubectl logs -n $(K8S_NAMESPACE) -l app=$(K8S_DEPLOYMENT) --tail=100 --all-containers 2>/dev/null || true

# Kind 集群
kind-create:
	kind create cluster --name $(KIND_NAME) --config kind-config.yaml

kind-up:
	-kind delete cluster --name $(KIND_NAME)
	$(MAKE) kind-create
	$(MAKE) kind-load-deps
	$(MAKE) k8s-all

kind-down:
	-kind delete cluster --name $(KIND_NAME)
