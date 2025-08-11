# TodoIng Backend Makefile
# Complete build and development tasks for proto-based architecture

# Variables
GO_VERSION := 1.23
PROTO_ROOT := api/proto
PROTO_DIR := $(PROTO_ROOT)/v1
GENERATED_DIR := pkg/api/v1
BINARY_NAME := server
GRPC_BINARY_NAME := grpc-server
DOCKER_IMAGE := todoing-backend

# Proto generation tools
PROTOC := protoc
PROTOC_GEN_GO := protoc-gen-go
PROTOC_GEN_GO_GRPC := protoc-gen-go-grpc

# Default target
.PHONY: all
all: clean deps proto build

# Help target
.PHONY: help
help:
	@echo "TodoIng Backend Makefile Commands:"
	@echo ""
	@echo "Development:"
	@echo "  install-tools    Install protobuf tools and dependencies"
	@echo "  deps            Download Go dependencies"
	@echo "  proto           Generate Go code from proto files"
	@echo "  openapi         Generate OpenAPI swagger from proto"
	@echo "  proto-all       Generate proto + OpenAPI"
	@echo "  build           Build HTTP server binary"
	@echo "  build-grpc      Build gRPC server binary"
	@echo "  run             Run HTTP server locally"
	@echo "  run-grpc        Run gRPC server locally"
	@echo ""
	@echo "Documentation:"
	@echo "  openapi         Generate OpenAPI (swagger) JSON"
	@echo "  docs-openapi    Proto -> OpenAPI + summary list"
	@echo "  docs-clean      Clean swagger output"
	@echo ""
	@echo "Testing:"
	@echo "  test            Run all tests"
	@echo "  test-unit       Run unit tests"
	@echo "  test-convert    Run conversion layer tests"
	@echo "  test-coverage   Run tests with coverage report"
	@echo ""
	@echo "Quality:"
	@echo "  lint            Run Go linter"
	@echo "  fmt             Format Go code"
	@echo "  vet             Run Go vet"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build         Build local Docker image"
	@echo "  docker-build-official Build official Docker image"
	@echo "  docker-run           Run local Docker container"
	@echo "  docker-run-official  Run official Docker container"
	@echo "  docker-pull          Pull official Docker image"
	@echo "  docker-stop          Stop and remove Docker containers"
	@echo "  docker-dev           Start Docker development environment"
	@echo ""
	@echo "Utilities:"
	@echo "  clean           Clean build artifacts"
	@echo "  clean-proto     Clean generated proto files"

# Install protobuf tools
.PHONY: install-tools
install-tools:
	@echo "Ensuring go.mod go version format..."
	@grep '^go 1.23$$' go.mod >/dev/null || sed -i '' 's/^go .*/go 1.23/' go.mod || true
	@echo "Installing protobuf tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.26.2
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.26.2
	@echo "Protobuf tools installed successfully!"

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading Go dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies downloaded!"

# Generate Go code from proto files
.PHONY: proto
PROTO_IMPORTS := -I $(PROTO_DIR) -I third_party
proto: install-tools
	@echo "Generating Go code (all protos in one invocation)..."
	@mkdir -p $(GENERATED_DIR)
	@PROTO_TOOLS_BIN=$$(go env GOPATH)/bin; \
	PATH="$$PROTO_TOOLS_BIN:$$PATH" protoc $(PROTO_IMPORTS) \
	  --go_out=$(GENERATED_DIR) --go_opt=paths=source_relative \
	  --go-grpc_out=$(GENERATED_DIR) --go-grpc_opt=paths=source_relative \
	  --grpc-gateway_out=$(GENERATED_DIR) --grpc-gateway_opt=paths=source_relative,generate_unbound_methods=true \
	  $(PROTO_DIR)/*.proto || exit 1; \
	echo "Proto code generation completed!"

# OpenAPI (Swagger) generation from proto
.PHONY: openapi
openapi: install-tools
	@echo "Generating OpenAPI (swagger) from proto files..."
	@PROTO_TOOLS_BIN=$$(go env GOPATH)/bin; \
	mkdir -p docs/swagger; \
	PATH="$$PROTO_TOOLS_BIN:$$PATH" protoc $(PROTO_IMPORTS) \
	  --openapiv2_out=docs/swagger \
	  --openapiv2_opt=allow_merge=true,merge_file_name=todoing,logtostderr=true,simple_operation_ids=true,json_names_for_fields=false \
	  $(PROTO_DIR)/*.proto; \
	echo "Swagger JSON: docs/swagger/todoing.swagger.json"

# Convenience target: proto + openapi
.PHONY: proto-all
proto-all: proto openapi
	@echo "Proto + OpenAPI generation completed."

# Build HTTP server
.PHONY: build
build: deps
	@echo "Building HTTP server..."
	go build -ldflags="-w -s" -o $(BINARY_NAME) ./cmd/api/main.go
	@echo "HTTP server built: $(BINARY_NAME)"

# Build gRPC server
.PHONY: build-grpc
build-grpc: deps proto
	@echo "Building gRPC server..."
	go build -ldflags="-w -s" -o $(GRPC_BINARY_NAME) ./cmd/grpc/main.go
	@echo "gRPC server built: $(GRPC_BINARY_NAME)"

# Run HTTP server locally
.PHONY: run
run: build
	@echo "Starting HTTP server..."
	./$(BINARY_NAME)

# Run gRPC server locally
.PHONY: run-grpc
run-grpc: build-grpc
	@echo "Starting gRPC server..."
	./$(GRPC_BINARY_NAME)

# Run all tests
.PHONY: test
test:
	@echo "Running all tests..."
	go test -v ./...

# Run unit tests only
.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	go test -v -short ./...

# Run conversion layer tests
.PHONY: test-convert
test-convert:
	@echo "Running conversion layer tests..."
	go test -v ./internal/convert/...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint code
.PHONY: lint
lint:
	@echo "Running Go linter..."
	golangci-lint run

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

# Vet code
.PHONY: vet
vet:
	@echo "Running Go vet..."
	go vet ./...

# Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):latest .
	@echo "Docker image built: $(DOCKER_IMAGE):latest"

# Build Docker image with official tag
.PHONY: docker-build-official
docker-build-official:
	@echo "Building official Docker image..."
	docker build -t axiu/todoing-go:latest -t axiu/todoing-go:dev .
	@echo "Official Docker image built: axiu/todoing-go:latest"

# Run Docker container
.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run -d --name todoing-go-backend \
		-p 5004:5004 \
		-e MONGO_URI="mongodb://localhost:27017/todoing" \
		-e JWT_SECRET="development_jwt_secret_key" \
		$(DOCKER_IMAGE):latest
	@echo "Docker container started: todoing-go-backend"

# Run official Docker image
.PHONY: docker-run-official
docker-run-official:
	@echo "Running official Docker image..."
	docker run -d --name todoing-go-official \
		-p 5004:5004 \
		-e MONGO_URI="mongodb://localhost:27017/todoing" \
		-e JWT_SECRET="development_jwt_secret_key" \
		axiu/todoing-go:latest
	@echo "Official Docker container started: todoing-go-official"

# Pull official Docker image
.PHONY: docker-pull
docker-pull:
	@echo "Pulling official Docker image..."
	docker pull axiu/todoing-go:latest
	@echo "Official Docker image pulled: axiu/todoing-go:latest"

# Stop and remove Docker containers
.PHONY: docker-stop
docker-stop:
	@echo "Stopping Docker containers..."
	-docker stop todoing-go-backend todoing-go-official 2>/dev/null || true
	-docker rm todoing-go-backend todoing-go-official 2>/dev/null || true
	@echo "Docker containers stopped and removed"

# Docker development environment
.PHONY: docker-dev
docker-dev:
	@echo "Starting Docker development environment..."
	cd .. && ./docker/deploy.sh golang up --build
	@echo "Docker development environment started"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -f $(GRPC_BINARY_NAME)
	rm -f coverage.out
	rm -f coverage.html
	@echo "Clean completed!"

# Clean generated proto files
.PHONY: clean-proto
clean-proto:
	@echo "Cleaning generated proto files..."
	rm -rf $(GENERATED_DIR)
	@echo "Proto files cleaned!"

# Quick development setup
.PHONY: dev-setup
dev-setup: install-tools deps proto
	@echo "Development setup completed!"
	@echo "You can now run 'make build' or 'make build-grpc'"

# Verify installation
.PHONY: verify
verify:
	@echo "Verifying installation..."
	@echo "Go version: $(shell go version)"
	@echo "Protoc version: $(shell protoc --version || echo 'protoc not found')"
	@echo "protoc-gen-go: $(shell which protoc-gen-go || echo 'not found')"
	@echo "protoc-gen-go-grpc: $(shell which protoc-gen-go-grpc || echo 'not found')"
	@echo ""
	@echo "Proto files:"
	@find $(PROTO_DIR) -name "*.proto" | head -5
	@echo ""
	@echo "Generated files:"
	@find $(GENERATED_DIR) -name "*.go" 2>/dev/null | head -5 || echo "No generated files found"

# Documentation generation
.PHONY: docs-clean
docs-clean:
	@echo "Cleaning swagger documentation..."
	rm -f docs/swagger/todoing.swagger.json || true
	@echo "Swagger documentation cleaned!"

# Proto -> OpenAPI + summary
.PHONY: docs-openapi
docs-openapi: openapi
	@echo "===> OpenAPI 主文件: docs/swagger/todoing.swagger.json" || true
	@ls -1 docs/swagger | sed 's/^/ - /'
