BINARY_NAME=mnemo
BUILD_DIR=./dist
CMD_PATH=./cmd/mnemo
FRONTEND_DIR=./internal/dashboard/frontend

IMAGE=ghcr.io/kyungw00k/mnemo

.PHONY: build build-linux-static frontend-build frontend-dev test lint clean run-stdio run-sse install install-local docker-build docker-run docker-push dev

frontend-build:
	npm install --prefix $(FRONTEND_DIR)
	npm run build --prefix $(FRONTEND_DIR)

build: frontend-build
	CGO_ENABLED=1 go build -tags sqlite_fts5 -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

build-linux-static: frontend-build
	CGO_ENABLED=1 GOOS=linux go build -tags sqlite_fts5 -ldflags="-extldflags=-static" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-static $(CMD_PATH)

test:
	CGO_ENABLED=1 go test -tags sqlite_fts5 ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(FRONTEND_DIR)/dist
	rm -rf $(FRONTEND_DIR)/node_modules

# dev: start Go SSE server and Vite dev server concurrently
dev:
	@echo "Starting mnemo SSE + Vite dev server..."
	@TRANSPORT=sse CGO_ENABLED=1 go run -tags sqlite_fts5 $(CMD_PATH) &
	@npm run dev --prefix $(FRONTEND_DIR)

run-stdio:
	TRANSPORT=stdio go run $(CMD_PATH)

run-sse:
	TRANSPORT=sse go run $(CMD_PATH)

install:
	CGO_ENABLED=1 go install -tags sqlite_fts5 $(CMD_PATH)

install-local:
	mkdir -p $(HOME)/.local/bin
	CGO_ENABLED=1 go build -tags sqlite_fts5 -o $(HOME)/.local/bin/$(BINARY_NAME) $(CMD_PATH)

docker-build:
	docker build -t mnemo:latest .

docker-push:
	docker buildx build --platform linux/amd64,linux/arm64 \
		-t $(IMAGE):dev \
		--push .

docker-run:
	docker run -i --rm \
		-e DB_URL=$(DB_URL) \
		-e EMBEDDING_BASE_URL=$(EMBEDDING_BASE_URL) \
		-e EMBEDDING_API_KEY=$(EMBEDDING_API_KEY) \
		-e EMBEDDING_MODEL=$(EMBEDDING_MODEL) \
		-e TRANSPORT=$(or $(TRANSPORT),both) \
		-p 8765:8765 \
		mnemo:latest

all: build
