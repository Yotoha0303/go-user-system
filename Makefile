APP_NAME := go-user-system
IMAGE_NAME := go-user-system:dev

# Kubernetes
K8S_DIR := k8s
K8S_NAMESPACE := go-user-system
K8S_DEPLOYMENT := go-user-system
K8S_IMAGE ?= $(APP_NAME):latest
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

.PHONY: help run test coverage coverage-html integration-test race-test vet lint lint-fix swagger callvis callvis-serve \
	build build-windows build-linux clean tidy \
	goose-version migrate-create migrate-validate migrate-status migrate-version migrate-up migrate-up-by-one migrate-down migrate-redo migrate-reset migrate-fix \
	docker-build compose-up compose-down compose-logs ci \
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
	go build -o bin/$(APP_NAME) ./cmd

build-windows: export GOOS=windows
build-windows:
	go build -o bin/$(APP_NAME).exe ./cmd

build-linux: export CGO_ENABLED=0
build-linux: export GOOS=linux
build-linux:
	go build -o bin/$(APP_NAME) ./cmd

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
	docker build -t $(IMAGE_NAME) .

compose-up:
	docker compose up -d --build

compose-down:
	docker compose down

compose-logs:
	docker compose logs -f app

ci:
	$(MAKE) lint
	$(MAKE) test
	$(MAKE) race-test
	$(MAKE) vet
	$(MAKE) build
	$(MAKE) docker-build

# Kubernetes
k8s-namespace:
	kubectl create namespace $(K8S_NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -

k8s-build:
	docker build --platform linux/amd64 -t $(K8S_IMAGE) .

k8s-build-kind: k8s-build
	kind load docker-image $(K8S_IMAGE) --name $(KIND_NAME)

# Bypass "kind load" for multi-arch images. kind uses --all-platforms
# which fails when Docker only has linux/amd64 blobs.
kind-load-deps:
	docker pull --platform linux/amd64 mysql:8.4
	docker pull --platform linux/amd64 redis:7.4-alpine
	docker pull --platform linux/amd64 nginx:alpine
	docker save mysql:8.4 | docker exec -i $(KIND_NAME)-control-plane ctr -n k8s.io images import --platform linux/amd64 --base-name docker.io/library/mysql:8.4 -
	docker save redis:7.4-alpine | docker exec -i $(KIND_NAME)-control-plane ctr -n k8s.io images import --platform linux/amd64 --base-name docker.io/library/redis:7.4-alpine -
	docker save nginx:alpine | docker exec -i $(KIND_NAME)-control-plane ctr -n k8s.io images import --platform linux/amd64 --base-name docker.io/library/nginx:alpine -

k8s-build-push: k8s-build
	docker push $(K8S_IMAGE)

k8s-deploy: k8s-namespace
	kubectl apply -f $(K8S_DIR)/ --recursive

k8s-undeploy:
	kubectl delete -f $(K8S_DIR)/ --recursive --ignore-not-found=true

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
