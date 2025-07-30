.PHONY: build run test clean docker-build docker-run streamlit setup

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=basis-trader
BINARY_PATH=./bin/$(BINARY_NAME)

# Build the application
build:
	@echo "Building basis trader..."
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY_PATH) -v ./cmd/trader

# Run the application
run: build
	@echo "Running basis trader..."
	$(BINARY_PATH)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f $(BINARY_NAME)

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Setup development environment
setup:
	@echo "Setting up development environment..."
	@cp .env.example .env
	@mkdir -p data
	@echo "Please edit .env file with your Coinbase API credentials"
	@echo "Installing Python dependencies for Streamlit..."
	@cd streamlit && pip install -r requirements.txt

# Run Streamlit dashboard
streamlit:
	@echo "Starting Streamlit dashboard..."
	@cd streamlit && streamlit run app.py

# Run both services
run-all:
	@echo "Starting all services..."
	@make run &
	@make streamlit

# Docker commands
docker-build:
	@echo "Building Docker image..."
	@docker build -t basis-trader:latest .

docker-run:
	@echo "Running Docker container..."
	@docker run -p 8080:8080 -p 8501:8501 --env-file .env basis-trader:latest

# Development helpers
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

lint:
	@echo "Running linter..."
	@golangci-lint run

# Generate mocks for testing
mocks:
	@echo "Generating mocks..."
	@mockgen -source=pkg/coinbase/client.go -destination=pkg/coinbase/mock_client.go -package=coinbase