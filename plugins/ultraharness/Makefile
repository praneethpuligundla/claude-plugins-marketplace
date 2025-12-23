# Ultraharness Go Hooks Build System
# Cross-compiles hooks for macOS (arm64/amd64), Linux (amd64), and Windows (amd64)

HOOKS := pre_tool_use post_tool_use session_start user_prompt_submit subagent_stop pre_compact stop
UNIX_PLATFORMS := darwin-arm64 darwin-amd64 linux-amd64
WIN_PLATFORMS := windows-amd64

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

# Build all platforms for distribution (Unix-like + Windows with .exe)
all: $(foreach p,$(UNIX_PLATFORMS),$(foreach h,$(HOOKS),bin/$(p)/$(h))) \
     $(foreach h,$(HOOKS),bin/windows-amd64/$(h).exe) \
     bin/run-hook

# Platform auto-detection wrapper script
bin/run-hook:
	@mkdir -p bin
	@echo '#!/bin/bash' > $@
	@echo '# Auto-detect platform and run the appropriate Go binary hook' >> $@
	@echo 'set -e' >> $@
	@echo 'HOOK_NAME="$$1"' >> $@
	@echo 'shift 2>/dev/null || true' >> $@
	@echo 'SCRIPT_DIR="$$(cd "$$(dirname "$$0")" && pwd)"' >> $@
	@echo 'PLUGIN_ROOT="$$(dirname "$$SCRIPT_DIR")"' >> $@
	@echo 'OS_RAW="$$(uname -s)"' >> $@
	@echo 'ARCH="$$(uname -m)"' >> $@
	@echo 'case "$$OS_RAW" in' >> $@
	@echo '    Darwin) OS="darwin" ;;' >> $@
	@echo '    Linux) OS="linux" ;;' >> $@
	@echo '    MINGW*|CYGWIN*|MSYS*) OS="windows" ;;' >> $@
	@echo '    *) exec python3 "$${PLUGIN_ROOT}/hooks/$${HOOK_NAME}.py" "$$@" ;;' >> $@
	@echo 'esac' >> $@
	@echo 'case "$$ARCH" in' >> $@
	@echo '    x86_64|amd64) ARCH="amd64" ;;' >> $@
	@echo '    arm64|aarch64) ARCH="arm64" ;;' >> $@
	@echo '    *) exec python3 "$${PLUGIN_ROOT}/hooks/$${HOOK_NAME}.py" "$$@" ;;' >> $@
	@echo 'esac' >> $@
	@echo 'PLATFORM="$${OS}-$${ARCH}"' >> $@
	@echo 'if [ "$$OS" = "windows" ]; then' >> $@
	@echo '    GO_BINARY="$${SCRIPT_DIR}/$${PLATFORM}/$${HOOK_NAME}.exe"' >> $@
	@echo 'else' >> $@
	@echo '    GO_BINARY="$${SCRIPT_DIR}/$${PLATFORM}/$${HOOK_NAME}"' >> $@
	@echo 'fi' >> $@
	@echo 'if [ -x "$$GO_BINARY" ]; then' >> $@
	@echo '    exec "$$GO_BINARY" "$$@"' >> $@
	@echo 'else' >> $@
	@echo '    PYTHON_HOOK="$${PLUGIN_ROOT}/hooks/$${HOOK_NAME}.py"' >> $@
	@echo '    if [ -f "$$PYTHON_HOOK" ]; then' >> $@
	@echo '        exec python3 "$$PYTHON_HOOK" "$$@"' >> $@
	@echo '    else' >> $@
	@echo '        echo "{\"systemMessage\": \"[Harness] Hook not found: $$HOOK_NAME\"}"' >> $@
	@echo '        exit 0' >> $@
	@echo '    fi' >> $@
	@echo 'fi' >> $@
	@chmod +x $@

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

# Windows AMD64 (requires .exe extension)
bin/windows-amd64/%.exe: cmd/%/main.go
	@mkdir -p bin/windows-amd64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $@ ./cmd/$*

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
