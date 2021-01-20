# Variable
COMMIT=$(shell git rev-parse HEAD)
VERSION=$(shell git describe --tags --abbrev=0)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

.PHONY: golang

proto: protocol/astris.proto
	@echo Building Astris GRPC Code
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
  		protocol/astris.proto

# go does a pretty good job of managing it's iterative builds
compile: proto */*.go main.go
	# remember ldflags for setting Version and Date
	go build -o bin/astris -ldflags "-X main.Version=" main.go

run: golang
	bin/astris