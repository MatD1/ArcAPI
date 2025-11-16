.PHONY: build run test clean docker-up docker-down migrate build-frontend

# Build the application
build: build-frontend
	go build -o server ./cmd/server

# Build frontend
build-frontend:
	cd frontend && npm install && npm run build
	@echo "Frontend built successfully"

# Run the application
run:
	go run ./cmd/server

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -f server
	rm -f coverage.out
	rm -rf frontend/out
	rm -rf frontend/.next
	rm -rf frontend/node_modules

# Start Docker services
docker-up:
	docker-compose up -d

# Stop Docker services
docker-down:
	docker-compose down

# Start Docker services with logs
docker-logs:
	docker-compose up

# Download dependencies
deps:
	go mod download
	go mod tidy
	cd frontend && npm install

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Migrate database (if using migrations)
migrate:
	@echo "Note: GORM auto-migrates on startup. See migrations/ for SQL reference."

# Generate GraphQL code
generate-graphql:
	@echo "Generating GraphQL code..."
	cd internal/graph && go run github.com/99designs/gqlgen generate
	@echo "GraphQL code generated successfully!"
	@echo "Note: You may need to update internal/graph/handler.go after generation"

# Run full setup
setup: deps
	@echo "Dependencies downloaded. Don't forget to:"
	@echo "1. Copy .env.example to .env"
	@echo "2. Update .env with your configuration"
	@echo "3. Start PostgreSQL and Redis (docker-compose up -d postgres redis)"
	@echo "4. Build frontend: make build-frontend"
	@echo "5. Generate GraphQL code: make generate-graphql"
