BINARY_NAME=nocti
INSTALL_DIR=$(HOME)/.local/bin

.PHONY: all build install clean help

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o build/$(BINARY_NAME) main.go

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@mv build/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
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
	@rm -f $(BINARY_NAME)
	@rm -f nocti.json

help:
	@echo "Usage:"
	@echo "  make build    - Build the binary"
	@echo "  make install  - Build and install the binary to $(INSTALL_DIR)"
	@echo "  make clean    - Remove build artifacts"
	@echo "  make help     - Show this help message"
