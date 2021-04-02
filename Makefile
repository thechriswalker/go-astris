# Variable
COMMIT=$(shell git rev-parse HEAD)
VERSION=$(shell git describe --tags --abbrev=0)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
PKG = github.com/thechriswalker/go-astris
LDFLAGS := "-X $(PKG)/astris.Version=$(VERSION) -X $(PKG)/astris.Commit=$(COMMIT) -X $(PKG)/astris.BuildDate=$(BUILD_DATE)"
BUILD_DIR=build
ifeq ($(NO_EMBED),1)
    BUILD_TAGS := "noembed"
else
    BUILD_TAGS := " "
endif


# go does a pretty good job of managing it's iterative builds
$(BUILD_DIR)/astris: \
					protocol/astris_grpc.pb.go \
					main.go \
					astris/*.go \
					blockchain/*.go \
					cmds/*/*.go \
					crypto/*/*.go \
					ui/*.go \
					ui/assets/* \
					ui/build/*/*.js
	@echo ">>> Building Go source..."
	go build -tags $(BUILD_TAGS) -o $(BUILD_DIR)/astris -ldflags $(LDFLAGS) main.go

ui/build/*/*.js:
	cd ui && yarn build

protocol/astris_grpc.pb.go: protocol/astris.proto
	@echo ">>> Codegen for protobuf/GRPC..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
  		protocol/astris.proto

run: $(BUILD_DIR)/astris
	@$(BUILD_DIR)/astris