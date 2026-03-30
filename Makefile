.PHONY: setup install stop dev

setup:
	@echo "Setting up petshop environment..."
	@mkdir -p bin

install:
	@echo "Installing petshop dependencies..."

stop:
	@echo "Stopping petshop services..."

dev:
	@echo "Starting petshop in dev mode..."