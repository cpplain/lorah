.PHONY: help build clean test fmt lint check install

# Default target - show help
help:
	@echo "lorah - Harness for Long-Running Autonomous Coding Agents"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  help           Show this help message"
	@echo "  build          Build the lorah binary"
	@echo "  install        Install lorah to GOPATH/bin"
	@echo "  clean          Remove build artifacts"
	@echo "  test           Run all tests with race detector"
	@echo "  fmt            Format all Go code"
	@echo "  lint           Run go vet for static analysis"
	@echo "  check          Run fmt, test, and lint in sequence"

# Build the lorah binary
build:
	@# Generate dev+timestamp for local builds (releases override via ldflags)
	@VERSION=$$(date -u '+dev+%Y%m%d%H%M%S'); \
	echo "Building lorah $$VERSION..."; \
	go build -ldflags "-X 'main.Version=$$VERSION'" -o lorah cmd/lorah/main.go

# Install lorah to GOPATH/bin
install:
	@VERSION=$$(date -u '+dev+%Y%m%d%H%M%S'); \
	echo "Installing lorah $$VERSION..."; \
	go install -ldflags "-X 'main.Version=$$VERSION'" ./cmd/lorah

# Clean build artifacts
clean:
	rm -f lorah

# Run tests with race detector
test:
	go test -race ./...

# Format code
fmt:
	@if command -v goimports >/dev/null 2>&1; then \
		echo "Running goimports..."; \
		goimports -w .; \
	else \
		echo "goimports not found, using gofmt..."; \
		gofmt -w .; \
	fi

# Run static analysis with go vet
lint:
	@echo "Running go vet..."
	go vet ./...

# Run all checks
check: fmt test lint
