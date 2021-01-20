# Variable
COMMIT=$(shell git rev-parse HEAD)
VERSION=$(shell git describe --tags --abbrev=0)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)"

protocol/astris_grpc.pb.go: protocol/astris.proto
	@echo ">>> Codegen for protobuf/GRPC..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
  		protocol/astris.proto

# go does a pretty good job of managing it's iterative builds
bin/astris: protocol/astris_grpc.pb.go main.go astris/*.go blockchain/*.go
	@echo ">>> Building Go source..."
	go build -o bin/astris -ldflags $(LDFLAGS) main.go

run: bin/astris
	@bin/astris