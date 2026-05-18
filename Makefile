.PHONY: build test install

BINARY ?= mailcli

build:
	go build -o $(BINARY) ./cmd/mailcli

test:
	go test ./...

install:
	go install ./cmd/mailcli
