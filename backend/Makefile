.PHONY: all build clean

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

.PHONY: $(BINARIES)
