.PHONY: all build clean proto

BINARY_DIR := bin
CMD_DIRS := $(wildcard cmd/*)
BINARIES := $(patsubst cmd/%,%,$(CMD_DIRS))

all: build

build: $(BINARIES)

$(BINARIES): %:
	@echo "Building $@..."
	@mkdir -p $(BINARY_DIR)
	@go build -o $(BINARY_DIR)/$@ ./cmd/$@

clean:
	rm -rf $(BINARY_DIR)

list:
	@echo "Available binaries:"
	@echo "$(BINARIES)" | tr ' ' '\n' | sed 's/^/  - /'

proto:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/clickhouse.proto
	@echo "Protobuf code generated successfully"

.PHONY: $(BINARIES)
