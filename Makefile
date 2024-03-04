.PHONY: all build clean test lint ci

# Binary name for the output binary
BINARY_NAME=fork-sweeper

# Default command to run when no arguments are provided to make
all: build

# Builds the binary
build:
	@echo "Building..."
	go build -C cmd/fork-sweeper -o ../../${BINARY_NAME}

# Cleans our project: deletes binaries
clean:
	@echo "Cleaning..."
	go clean
	rm -f ${BINARY_NAME}

# Runs tests
test:
	@echo "Running tests..."
	go test ./... -cover

# Lints the project
lint:
	@echo "Linting..."
	go fmt ./...
	go vet ./...
	go mod tidy

# Command for Continuous Integration
ci: lint test
	@echo "CI steps..."
	# Add commands specific to your CI setup
	# e.g., integration testing, deployment commands, etc.

# Additional commands can be added below for database migrations, Docker operations, etc.
