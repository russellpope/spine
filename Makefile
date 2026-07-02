BIN := $(HOME)/bin

.PHONY: build test install

build:
	go build -o bin/spine ./cmd/spine

test:
	go test ./...

install:
	mkdir -p $(BIN)
	go build -o $(BIN)/spine ./cmd/spine
