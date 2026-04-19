BINARY_NAME=nocti
BUILD_DIR=build
INSTALL_DIR=$(HOME)/.local/bin
TARGET_BINARY=$(BUILD_DIR)/$(BINARY_NAME)

VERSION=$(shell git describe --tags --always 2>/dev/null || echo "development")
LD_FLAGS="-X nocti/cmd.Version=$(VERSION)"

.PHONY: all build install clean test help

all: build

build:
	@echo "Building $(BINARY_NAME) $(VERSION) into $(BUILD_DIR)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags $(LD_FLAGS) -o $(TARGET_BINARY) main.go

test:
	@echo "Running tests..."
	go test -v ./...

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@mv $(TARGET_BINARY) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installation complete!"
	@case ":$(PATH):" in \
		*":$(INSTALL_DIR):"*) ;; \
		*) \
			echo ""; \
			echo "Reminder: Ensure $(INSTALL_DIR) is in your PATH."; \
			echo "You can add it by adding this line to your .bashrc or .zshrc:"; \
			echo 'export PATH="$(INSTALL_DIR):$$PATH"'; \
			;; \
	esac

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@rm -f nocti.json

help:
	@echo "Usage:"
	@echo "  make build    - Build the binary in $(BUILD_DIR)/"
	@echo "  make test     - Run unit tests"
	@echo "  make install  - Build and move the binary to $(INSTALL_DIR)"
	@echo "  make clean    - Remove build artifacts"
	@echo "  make help     - Show this help message"
