# Ultraharness Go Hooks Build System
# Cross-compiles hooks for macOS (arm64/amd64) and Linux (amd64)

HOOKS := pre_tool_use post_tool_use session_start user_prompt_submit subagent_stop pre_compact stop
PLATFORMS := darwin-arm64 darwin-amd64 linux-amd64

# Build flags: strip debug info and symbols for smaller binaries
LDFLAGS := -ldflags="-s -w"

.PHONY: all clean test build-local

# Default: build for current platform only (faster for development)
build-local:
	@for hook in $(HOOKS); do \
		if [ -d "cmd/$$hook" ]; then \
			echo "Building $$hook..."; \
			go build $(LDFLAGS) -o bin/$$hook ./cmd/$$hook; \
		fi \
	done

# Build all platforms for distribution
all: $(foreach p,$(PLATFORMS),$(foreach h,$(HOOKS),bin/$(p)/$(h)))

# Darwin ARM64 (Apple Silicon)
bin/darwin-arm64/%: cmd/%/main.go
	@mkdir -p bin/darwin-arm64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $@ ./cmd/$*

# Darwin AMD64 (Intel Mac)
bin/darwin-amd64/%: cmd/%/main.go
	@mkdir -p bin/darwin-amd64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $@ ./cmd/$*

# Linux AMD64
bin/linux-amd64/%: cmd/%/main.go
	@mkdir -p bin/linux-amd64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $@ ./cmd/$*

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Show binary sizes
sizes:
	@echo "Binary sizes:"
	@find bin -type f -exec ls -lh {} \; 2>/dev/null || echo "No binaries built yet"
