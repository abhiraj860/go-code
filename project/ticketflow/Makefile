.DEFAULT_GOAL := help
SHELL := /bin/bash

GOBIN     := $(shell go env GOPATH)/bin
COMPOSE   := docker compose -f deploy/docker-compose.yml
BUF       := $(GOBIN)/buf

# Pinned so codegen is byte-identical on every machine and in CI.
BUF_VERSION              := v1.47.2
PROTOC_GEN_GO_VERSION    := v1.35.2
PROTOC_GEN_GRPC_VERSION  := v1.5.1
GOLANGCI_LINT_VERSION    := v1.62.2

##@ Setup

.PHONY: tools
tools: ## Install pinned dev tooling (buf, protoc plugins, linter) into GOPATH/bin
	@echo ">> installing tooling into $(GOBIN)"
	go install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GRPC_VERSION)
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@echo ">> done"

##@ Protobuf

.PHONY: proto
proto: ## Generate Go code from .proto files
	@cd proto && PATH="$(GOBIN):$$PATH" $(BUF) generate
	@echo ">> generated proto/gen"

.PHONY: proto-lint
proto-lint: ## Lint .proto files
	@cd proto && $(BUF) lint

.PHONY: proto-breaking
proto-breaking: ## Fail if protos changed incompatibly vs origin/main
	@cd proto && $(BUF) breaking --against '../../../.git#branch=main,subdir=project/ticketflow/proto'

##@ Local stack

.PHONY: pull
pull: ## Pre-download every image (~3.7GB first time); safe to run in background
	$(COMPOSE) --profile all pull

.PHONY: up
up: ## Start core stack: postgres, redis, kafka, mongo (Phases 1-3)
	$(COMPOSE) up -d --wait
	@$(MAKE) --no-print-directory topics
	@$(MAKE) --no-print-directory endpoints

.PHONY: up-search
up-search: ## Core + ElasticSearch (Phase 2+)
	$(COMPOSE) --profile search up -d --wait
	@$(MAKE) --no-print-directory topics
	@$(MAKE) --no-print-directory endpoints

.PHONY: up-aws
up-aws: ## Core + LocalStack (Phase 4+)
	$(COMPOSE) --profile aws up -d --wait
	@$(MAKE) --no-print-directory topics
	@$(MAKE) --no-print-directory endpoints

.PHONY: up-all
up-all: ## Everything, including ElasticSearch and LocalStack
	$(COMPOSE) --profile all up -d --wait
	@$(MAKE) --no-print-directory topics
	@$(MAKE) --no-print-directory endpoints

.PHONY: endpoints
endpoints: ## Print connection details for whatever is currently running
	@echo ""
	@echo ">> stack ready"
	@docker ps --filter "name=tf-" --format '{{.Names}} {{.Ports}}' \
		| awk '{ printf "   %-18s %s\n", $$1, $$2 }'
	@echo ""
	@echo "   postgres  user/pass/db: ticketflow (+ ticketflow_{catalog,inventory,order})"
	@echo "   mongo     user/pass: ticketflow"
	@echo ""

.PHONY: down
down: ## Stop the stack and delete its volumes
	$(COMPOSE) --profile all down -v

.PHONY: ps
ps: ## Show stack status
	$(COMPOSE) ps

.PHONY: logs
logs: ## Tail stack logs (make logs SVC=kafka to scope)
	$(COMPOSE) logs -f $(SVC)

.PHONY: topics
topics: ## Create Kafka topics (auto-create is disabled on purpose)
	@for t in catalog.event.updated order.created ticket.issued seat.released dlq; do \
		docker exec tf-kafka /opt/kafka/bin/kafka-topics.sh \
			--bootstrap-server localhost:9092 \
			--create --if-not-exists --topic $$t \
			--partitions 3 --replication-factor 1 >/dev/null 2>&1 \
			&& echo ">> topic $$t ready" || echo "!! topic $$t failed"; \
	done

##@ Run it

.PHONY: run
run: ## Build and start every service (needs `make up-all` first)
	@./scripts/dev.sh start

.PHONY: stop
stop: ## Stop every service
	@./scripts/dev.sh stop

.PHONY: seed
seed: ## Load demo events, seats and search index
	@./scripts/dev.sh seed

.PHONY: reset
reset: ## Release every seat and delete orders (keeps the catalogue)
	@./scripts/dev.sh reset

.PHONY: status
status: ## Show what is running
	@./scripts/dev.sh status

.PHONY: demo
demo: ## Scripted end-to-end walkthrough
	@./scripts/demo.sh

##@ Build & test

.PHONY: build
build: ## Compile every module in the workspace
	go build ./...
	cd pkg && go build ./...
	cd proto && go build ./...

.PHONY: test
test: ## Run Go unit tests across the workspace
	go test ./... $(TESTFLAGS)
	cd pkg && go test ./... $(TESTFLAGS)

.PHONY: test-node
test-node: ## Typecheck and test the Node packages
	cd node/realtime-gateway && npm run typecheck && npm run build && node --test dist/hub.test.js

.PHONY: test-all
test-all: test test-node ## Everything CI runs, minus the web build

.PHONY: lint
lint: ## Run go vet and golangci-lint
	go vet ./...
	cd pkg && go vet ./...
	@command -v $(GOBIN)/golangci-lint >/dev/null 2>&1 \
		&& $(GOBIN)/golangci-lint run ./... \
		|| echo ">> golangci-lint not installed, run 'make tools'"

.PHONY: tidy
tidy: ## Tidy every module
	go mod tidy
	cd pkg && go mod tidy
	cd proto && go mod tidy

##@ Meta

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nTicketFlow\n\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
	@echo ""
