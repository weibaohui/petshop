.PHONY: setup install stop dev build run

setup:
	@echo "Setting up petshop environment..."
	@mkdir -p bin

install:
	@echo "Installing petshop dependencies..."
	go mod download

stop:
	@echo "Stopping petshop services..."
	@pkill -f "petshop" 2>/dev/null || true

dev:
	@echo "Starting petshop in dev mode..."
	@go run cmd/server/main.go

build:
	@echo "Building petshop..."
	go build -o bin/petshop cmd/server/main.go

run:
	@echo "Running petshop..."
	@./bin/petshop