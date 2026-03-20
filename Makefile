.PHONY: help build clean fmt lint test

# Default target - show help
help:
	@echo "lorah - Harness for Long-Running Autonomous Coding Agents"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  help           Show this help message"
	@echo "  build          Build the lorah binary"
	@echo "  clean          Remove build artifacts"
	@echo "  fmt            Format all Go code"
	@echo "  lint           Run go vet for static analysis"
	@echo "  test           Run all tests"

# Build the lorah binary
build:
	@# Generate dev+timestamp for local builds (releases override via ldflags)
	@VERSION=$$(date -u '+dev+%Y%m%d%H%M%S'); \
	echo "Building lorah $$VERSION..."; \
	go build -ldflags "-X 'main.Version=$$VERSION'" -o ./bin/lorah .

# Clean build artifacts
clean:
	rm -rf ./bin

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

# Run all tests
test:
	go test ./...
