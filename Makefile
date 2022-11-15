BUILD_DIR?=$(CURDIR)/build
BIN?=$(BUILD_DIR)/ghere

all: build test
.PHONY: all

build:
	go build -o $(BIN) ./cmd/ghere/
.PHONY: build

test:
	go test ./...
.PHONY: test

install:
	go install ./cmd/ghere/
.PHONY: install

lint:
	golangci-lint run
.PHONY: lint

clean:
	go clean
	rm -rf $(BUILD_DIR)
.PHONY: clean
