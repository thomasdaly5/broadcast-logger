.PHONY: build clean test run-server run-client

# Build both server and client
build:
	go build -o bin/server ./cmd/server
	go build -o bin/client ./cmd/client

# Clean build artifacts
clean:
	rm -rf bin/

# Run tests
test:
	go test ./...

# Run the server
run-server: build
	./bin/server

# Run the client
run-client: build
	./bin/client

# Install dependencies
deps:
	go mod download
	go mod tidy 