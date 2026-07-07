.PHONY: gen-proto test

gen-proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-go=$(shell go tool -n google.golang.org/protobuf/cmd/protoc-gen-go) \
		--plugin=protoc-gen-go-grpc=$(shell go tool -n google.golang.org/grpc/cmd/protoc-gen-go-grpc) \
		proto/payment/v1/payment.proto

test:
	go test -v -race ./...