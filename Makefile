.PHONY: all fmt vet build run test clean install help

all: fmt vet test build

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

build: vet
	go build -o librarian ./cmd/librarian

run: build
	./librarian

test: vet
	go test -v ./...

clean:
	rm -f librarian
	go clean ./...

install: build
	mkdir -p ~/.local/bin
	cp librarian ~/.local/bin/

help:
	@echo "Available targets:"
	@echo "  all     - Format, vet, test, and build"
	@echo "  build   - Build the binary"
	@echo "  run     - Build and run"
	@echo "  test    - Run tests"
	@echo "  clean   - Remove build artifacts"
	@echo "  install - Install to ~/.local/bin"
