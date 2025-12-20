.PHONY: all build clean proto goclean modclean

BINARY_DIR := bin
CMD_DIRS := $(wildcard cmd/*)
BINARIES := $(patsubst cmd/%,%,$(CMD_DIRS))

all: build

build: $(BINARIES)

$(BINARIES): %:
	@echo "Building $@..."
	@mkdir -p $(BINARY_DIR)
	@go build -v -o $(BINARY_DIR)/$@ ./cmd/$@

clean:
	rm -rf $(BINARY_DIR)

goclean:
	@echo "Cleaning Go build cache, test cache and installed packages..."
	@go clean -cache -testcache || true
	@go clean -i ./... || true
	@echo "Go build/test caches and installed packages cleaned"

modclean:
	@echo "Cleaning Go module cache..."
	@go clean -modcache || true
	@echo "Module cache cleaned"

list:
	@echo "Available binaries:"
	@echo "$(BINARIES)" | tr ' ' '\n' | sed 's/^/  - /'

proto:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		internal/proto/can/can.proto
	@echo "Protobuf code generated successfully"

.PHONY: $(BINARIES)
