# tools/Makefile - Build and install scaffolding tools
#
# Usage:
#   make build       Build all tools
#   make install     Install tools to $GOPATH/bin
#   make test        Run tests
#   make clean       Remove build artifacts

.PHONY: all build install test clean help

# List of scaffold wrapper commands (delegate to GoTools)
SCAFFOLD_CMDS := add-endpoint add-model add-service add-middleware \
                 add-page add-component add-hook add-feature \
                 add-env-var sync-openapi

# List of utility commands (self-contained)
UTIL_CMDS := build-backend check-manifest check-requirement-coverage \
             create-app pre-flight update-from-template verify-hal-compliance

ALL_CMDS := $(SCAFFOLD_CMDS) $(UTIL_CMDS)

# Build output directory
BUILD_DIR := ./bin

all: build

build:
	@echo "Building tools..."
	@mkdir -p $(BUILD_DIR)
	@for cmd in $(ALL_CMDS); do \
		echo "  Building $$cmd..."; \
		go build -o $(BUILD_DIR)/$$cmd ./cmd/$$cmd; \
	done
	@echo "Done. Binaries in $(BUILD_DIR)/"

install:
	@echo "Installing tools..."
	@for cmd in $(ALL_CMDS); do \
		echo "  Installing $$cmd..."; \
		go install ./cmd/$$cmd; \
	done
	@echo "Done. Tools installed to $$GOPATH/bin/"

test:
	@echo "Running tests..."
	go test ./...

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)

# Verify tools compile
verify:
	@echo "Verifying all tools compile..."
	@for cmd in $(ALL_CMDS); do \
		go build -o /dev/null ./cmd/$$cmd 2>&1 || exit 1; \
	done
	@echo "All tools compile successfully."

# Maturity assessment targets
maturity:
	@go run github.com/DigiStratum/GoTools/cmd/assess-maturity@latest

maturity-json:
	@go run github.com/DigiStratum/GoTools/cmd/assess-maturity@latest --output json

maturity-gaps:
	@go run github.com/DigiStratum/GoTools/cmd/assess-maturity@latest --gaps

help:
	@echo "Available targets:"
	@echo "  build         - Build all tools to ./bin/"
	@echo "  install       - Install all tools to \$$GOPATH/bin/"
	@echo "  test          - Run tests"
	@echo "  clean         - Remove build artifacts"
	@echo "  verify        - Check all tools compile"
	@echo "  maturity      - Check current maturity level"
	@echo "  maturity-json - Output maturity assessment as JSON"
	@echo "  maturity-gaps - Show gaps to next maturity level"
	@echo ""
	@echo "Tools:"
	@echo "  Scaffold wrappers (delegate to ds-scaffold):"
	@for cmd in $(SCAFFOLD_CMDS); do \
		echo "    - $$cmd"; \
	done
	@echo ""
	@echo "  Utility tools (self-contained):"
	@for cmd in $(UTIL_CMDS); do \
		echo "    - $$cmd"; \
	done
