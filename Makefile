BINARY_NAME=jt
VERSION?=dev

.PHONY: build install test lint clean setup

build:
	go build -ldflags "-X github.com/erickhilda/jt/cmd.version=$(VERSION)" -o $(BINARY_NAME) .

install:
	go install -ldflags "-X github.com/erickhilda/jt/cmd.version=$(VERSION)" .

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY_NAME)

setup:
	git config core.hooksPath .githooks
