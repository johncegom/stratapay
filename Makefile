export PATH := $(shell go env GOPATH)/bin:$(PATH)

# Prefer Docker Compose v2 (`docker compose`) plugin; fall back to the
# standalone v1 (`docker-compose`) binary if v2 isn't available.
ifeq ($(shell docker compose version >/dev/null 2>&1 && echo yes),yes)
DOCKER_COMPOSE := docker compose
else
DOCKER_COMPOSE := docker-compose
endif

.PHONY: gen-proto test infra-up infra-down

gen-proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-go=$(shell go tool -n google.golang.org/protobuf/cmd/protoc-gen-go) \
		--plugin=protoc-gen-go-grpc=$(shell go tool -n google.golang.org/grpc/cmd/protoc-gen-go-grpc) \
		proto/payment/v1/payment.proto

infra-up:
	$(DOCKER_COMPOSE) up -d

infra-down:
	$(DOCKER_COMPOSE) down -v

test:
	go test -v -race -count=1 ./...