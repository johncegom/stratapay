# Ensure local go binaries are accessible
export PATH := $(shell go env GOPATH)/bin:$(PATH)

.PHONY: init gen-proto test

init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc

gen-proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/payment/v1/payment.proto

test:
	go test -v -race ./...