setup:
	go mod tidy

install:
	go build -o petshop ./cmd/server

stop:
	pkill -f petshop || true

dev:
	@echo "Starting petshop..."
	@go run ./cmd/server
