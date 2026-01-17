.PHONY: all fmt check vet build build-cli run test clean install install-cli ci help

all: fmt vet test build build-cli

fmt:
	go fmt ./...

check:
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Files not formatted:"; \
		gofmt -l .; \
		exit 1; \
	fi

vet:
	go vet ./...

build: fmt vet
	go build -o libraio ./cmd/libraio

build-cli: fmt vet
	go build -o libraio-cli ./cmd/libraio-cli

run: build
	./libraio

test:
	go test -v ./...

clean:
	rm -f libraio libraio-cli
	go clean ./...

install: build
	mkdir -p ~/.local/bin
	cp libraio ~/.local/bin/

install-cli: build-cli
	mkdir -p ~/.local/bin
	cp libraio-cli ~/.local/bin/

ci: check vet build test

help:
	@echo "Available targets:"
	@echo "  all         - Format, vet, test, and build all binaries"
	@echo "  fmt         - Format code"
	@echo "  check       - Check formatting (no changes)"
	@echo "  vet         - Run go vet"
	@echo "  build       - Build the TUI binary"
	@echo "  build-cli   - Build the CLI binary"
	@echo "  run         - Build and run TUI"
	@echo "  test        - Run tests"
	@echo "  ci          - Run CI checks locally"
	@echo "  clean       - Remove build artifacts"
	@echo "  install     - Install TUI to ~/.local/bin"
	@echo "  install-cli - Install CLI to ~/.local/bin"
