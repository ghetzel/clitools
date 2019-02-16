.PHONY: cmd build
.EXPORT_ALL_VARIABLES:

GO111MODULE     ?= on
LOCALS          := $(shell find . -type f -name '*.go' 2> /dev/null)

all: deps fmt build

deps:
	go get ./...
	-go mod tidy

fmt:
	gofmt -w $(LOCALS)
	go vet ./...

build:
	$(foreach tool,$(wildcard cmd/*),go build -o bin/$(shell basename $(tool)) cmd/$(shell basename $(tool))/*.go)