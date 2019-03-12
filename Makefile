.PHONY: all cmd build
.EXPORT_ALL_VARIABLES:

TOOLS           := $(wildcard cmd/*)
GO111MODULE     ?= on
LOCALS          := $(shell find . -type f -name '*.go' 2> /dev/null)

all: deps fmt build

deps:
	go get ./...
	-go mod tidy

fmt:
	gofmt -w $(LOCALS)
	go vet ./...

.PHONY: $(TOOLS)
$(TOOLS):
	go build -o $(subst cmd,bin,$(@)) $(@)/*.go

build: $(TOOLS)