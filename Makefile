.PHONY: all cmd build
.EXPORT_ALL_VARIABLES:

TOOLS           := $(subst cmd,bin,$(wildcard cmd/*))
GO111MODULE     ?= on
LOCALS          := $(shell find . -type f -name '*.go' 2> /dev/null)
REGISTRY        ?= registry.apps.gammazeta.net/
VERSION          = $(shell grep -Po "\d+\.\d+\.\d+" version.go)

all: deps fmt build

deps:
	go get ./...
	-go mod tidy

fmt:
	gofmt -w $(LOCALS)
	go vet ./...

.PHONY: $(TOOLS)
$(TOOLS):
	go build -o $(@) $(subst bin,cmd,$(@))/*.go

build: $(TOOLS)
	cp bin/* ~/lib/apps/clitools/linux/amd64/

contrib:
	mkdir contrib

contrib/rclone-1.50.2.deb: contrib
	curl -sSfLo contrib/rclone-1.50.2.deb 'https://downloads.rclone.org/v1.50.2/rclone-v1.50.2-linux-amd64.deb'

docker: contrib/rclone-1.50.2.deb
	@echo "Building Docker image for v$(VERSION)"
	@docker build --quiet . > .docker-build-id
	@echo "Docker image ID: `cat .docker-build-id`"
	docker tag `cat .docker-build-id` $(REGISTRY)ghetzel/clitools:$(VERSION)
	docker tag $(REGISTRY)ghetzel/clitools:$(VERSION) $(REGISTRY)ghetzel/clitools:latest
	docker push $(REGISTRY)ghetzel/clitools:$(VERSION)
	docker push $(REGISTRY)ghetzel/clitools:latest
