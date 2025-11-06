# Build the binary
build:
    go build -o bin/chex ./cmd/chex

# Run tests
test:
    go test -v ./...

# Format code
format:
    gofmt -w .

# Check formatting
format-check:
    @if [ -n "$(gofmt -l .)" ]; then \
        echo "Files need formatting:"; \
        gofmt -l .; \
        exit 1; \
    fi

# Run linter
lint:
    mise exec -- golangci-lint run

# Fix linting issues
lint-fix:
    mise exec -- golangci-lint run --fix

# Run chex locally
run *ARGS:
    go run ./cmd/chex {{ARGS}}

# Install locally
install:
    go install ./cmd/chex

# Clean build artifacts
clean:
    rm -rf bin/

# Run all checks (format-check + lint + test)
check: format-check lint test
