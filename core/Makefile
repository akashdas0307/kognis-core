.PHONY: build test lint clean fmt vet

VERSION ?= $(shell grep 'const version' cmd/kognis/main.go | awk -F'"' '{print $$2}')
BINARY = kognis

build:
	go build -o bin/$(BINARY) ./cmd/kognis

test:
	go test ./... -v -count=1

test-e2e: build
	cd $(CURDIR)/.. && pytest tests/e2e -v

lint: fmt vet
	@echo "lint: ok"

fmt:
	gofmt -w .
	goimports -w . 2>/dev/null || true

vet:
	go vet ./...

clean:
	rm -rf bin/

run: build
	./bin/$(BINARY)

deps:
	go mod download
	go mod tidy

all: deps build test lint