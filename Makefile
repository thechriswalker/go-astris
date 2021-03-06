# Variable
COMMIT=$(shell git rev-parse HEAD)
VERSION=$(shell git describe --tags --abbrev=0)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)"
BUILD_DIR=build

# go does a pretty good job of managing it's iterative builds
$(BUILD_DIR)/astris: protocol/astris_grpc.pb.go main.go astris/*.go blockchain/*.go cmds/*/*.go crypto/*/*.go
	@echo ">>> Building Go source..."
	go build -o $(BUILD_DIR)/astris -ldflags $(LDFLAGS) main.go

protocol/astris_grpc.pb.go: protocol/astris.proto
	@echo ">>> Codegen for protobuf/GRPC..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
  		protocol/astris.proto

run: $(BUILD_DIR)/astris
	@$(BUILD_DIR)/astris