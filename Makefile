BINARY_NAME=mnemo
BUILD_DIR=./dist
CMD_PATH=./cmd/mnemo

.PHONY: build test lint clean run-stdio run-sse install docker-build docker-run

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)

run-stdio:
	TRANSPORT=stdio go run $(CMD_PATH)

run-sse:
	TRANSPORT=sse go run $(CMD_PATH)

install:
	go install $(CMD_PATH)

docker-build:
	docker build -t mnemo:latest .

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
