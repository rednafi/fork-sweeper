.PHONY: all build clean test lint lint-check ci


BINARY_NAME=fork-sweeper

all: build

build:
	@echo "Building..."
	go build -C cmd/fork-sweeper -o ../../${BINARY_NAME}

clean:
	@echo "Cleaning..."
	go clean
	rm -f ${BINARY_NAME}

test:
	@echo "Running tests..."
	go test ./... -cover

lint:
	@echo "Linting..."
	go fmt ./...
	go vet ./...
	go mod tidy

lint-check:
	@echo "Checking lint..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Some files are not formatted. Please run 'gofmt -w .' on your code."; \
		exit 1; \
	fi
