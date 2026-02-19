.PHONY: build test clean install install-full completion run help

# Build variables
BINARY_NAME=raycoon
BUILD_DIR=bin
MAIN_PATH=cmd/raycoon/main.go

# Build flags
LDFLAGS=-ldflags "-s -w"
BUILD_FLAGS=$(LDFLAGS)

help: ## Show this help message
	@echo "Raycoon - V2Ray/Proxy CLI Client"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...

coverage: test ## Generate coverage report
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

install: ## Install the application
	@echo "Installing $(BINARY_NAME)..."
	@go install $(BUILD_FLAGS) $(MAIN_PATH)
	@echo "✓ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

completion: build ## Generate shell completion files
	@echo "Generating shell completions..."
	@mkdir -p $(BUILD_DIR)/completions
	@./$(BUILD_DIR)/$(BINARY_NAME) completion bash > $(BUILD_DIR)/completions/raycoon.bash
	@./$(BUILD_DIR)/$(BINARY_NAME) completion zsh > $(BUILD_DIR)/completions/_raycoon
	@./$(BUILD_DIR)/$(BINARY_NAME) completion fish > $(BUILD_DIR)/completions/raycoon.fish
	@echo "✓ Completions generated in $(BUILD_DIR)/completions/"

install-full: build completion ## Build + install binary + completions + xray
	@echo "Full installation..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "✓ Binary installed to /usr/local/bin/$(BINARY_NAME)"
	@mkdir -p $(HOME)/.config/raycoon $(HOME)/.local/share/raycoon $(HOME)/.cache/raycoon
	@# Install completions for detected shell
	@if [ "$(shell basename $$SHELL)" = "zsh" ]; then \
		sudo mkdir -p /usr/local/share/zsh/site-functions; \
		sudo cp $(BUILD_DIR)/completions/_raycoon /usr/local/share/zsh/site-functions/_raycoon; \
		echo "✓ Zsh completions installed"; \
	elif [ "$(shell basename $$SHELL)" = "bash" ]; then \
		mkdir -p $(HOME)/.local/share/bash-completion/completions; \
		cp $(BUILD_DIR)/completions/raycoon.bash $(HOME)/.local/share/bash-completion/completions/raycoon; \
		echo "✓ Bash completions installed"; \
	elif [ "$(shell basename $$SHELL)" = "fish" ]; then \
		mkdir -p $(HOME)/.config/fish/completions; \
		cp $(BUILD_DIR)/completions/raycoon.fish $(HOME)/.config/fish/completions/raycoon.fish; \
		echo "✓ Fish completions installed"; \
	fi
	@echo "✓ Full installation complete"

run: ## Run the application
	@go run $(MAIN_PATH)

run-tui: ## Run with TUI
	@go run $(MAIN_PATH) tui

lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Format complete"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies updated"

dev: ## Run in development mode
	@go run $(MAIN_PATH) --verbose

.DEFAULT_GOAL := help
