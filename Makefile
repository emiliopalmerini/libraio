.PHONY: all fmt check vet build run test clean install ci help

all: fmt vet test build

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
	go build -o librarian ./cmd/librarian

run: build
	./librarian

test:
	go test -v ./...

clean:
	rm -f librarian
	go clean ./...

install: build
	mkdir -p ~/.local/bin
	cp librarian ~/.local/bin/

ci: check vet build test

help:
	@echo "Available targets:"
	@echo "  all     - Format, vet, test, and build"
	@echo "  fmt     - Format code"
	@echo "  check   - Check formatting (no changes)"
	@echo "  vet     - Run go vet"
	@echo "  build   - Build the binary"
	@echo "  run     - Build and run"
	@echo "  test    - Run tests"
	@echo "  ci      - Run CI checks locally"
	@echo "  clean   - Remove build artifacts"
	@echo "  install - Install to ~/.local/bin"
