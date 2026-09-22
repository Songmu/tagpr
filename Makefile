CURRENT_REVISION = $(shell git rev-parse --short HEAD)
BUILD_LDFLAGS = "-s -w -X github.com/Songmu/tagpr.revision=$(CURRENT_REVISION)"
u := $(if $(update),-u)

.PHONY: deps
deps:
	go get ${u}
	go mod tidy

.PHONY: docs-deps
docs-deps:
	cd docs/site && hugo mod tidy

.PHONY: devel-deps
devel-deps:
	go install github.com/Songmu/gocredits/cmd/gocredits@v0.5.0

.PHONY: test
test:
	go test

.PHONY: build
build:
	go build -ldflags=$(BUILD_LDFLAGS) ./cmd/tagpr

.PHONY: install
install:
	go install -ldflags=$(BUILD_LDFLAGS) ./cmd/tagpr

.PHONY: prepare-release
prepare-release: devel-deps
	go mod tidy
	gocredits -w
	git update-index --add --remove -- go.mod go.sum CREDITS
