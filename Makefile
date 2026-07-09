SHELL := /bin/sh

APP := grokbuild-proxy
VERSION ?= dev
BIN_DIR ?= bin
GOFLAGS ?= -trimpath
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: all build run test test-race vet check clean docker-build release-snapshot

all: check build

build:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build $(GOFLAGS) \
		-ldflags="$(LDFLAGS)" \
		-o $(BIN_DIR)/$(APP) \
		./cmd/$(APP)

run:
	go run ./cmd/$(APP) -config config.yaml

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

check:
	test -z "$$(gofmt -l ./cmd ./internal)"
	go vet ./...
	go test -race ./...
	GOTOOLCHAIN=go1.26.5 go run golang.org/x/vuln/cmd/govulncheck@latest ./...

docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		-t $(APP):$(VERSION) \
		.

release-snapshot:
	PATH="$$(go env GOPATH)/bin:$$PATH" \
		go run github.com/goreleaser/goreleaser/v2@latest \
		release --snapshot --clean

clean:
	rm -rf $(BIN_DIR) dist
