.PHONY: all cmd build
.EXPORT_ALL_VARIABLES:

TOOLS           := $(subst cmd,bin,$(wildcard cmd/*))
GO111MODULE     ?= on
LOCALS          := $(shell find . -type f -name '*.go' 2> /dev/null)
REGISTRY        ?= registry.apps.gammazeta.net/
VERSION          = $(shell grep -Po "\d+\.\d+\.\d+" version.go)
CGO_ENABLED     ?= 0
DESTDIR         ?= $(HOME)/lib/apps/clitools/$(shell go env GOOS)/$(shell go env GOARCH)/
PATH            := $(PWD)/bin:$(PATH)

all: deps fmt build

deps:
	go get ./...
	-go mod tidy

fmt:
	gofmt -w $(LOCALS)
	go vet .

$(TOOLS):
	go build -ldflags="-s -w" -o $(@) $(subst bin,cmd,$(@))/*.go

build: $(TOOLS) deploy

deploy:
	@test -d "$(DESTDIR)" || mkdir -p "$(DESTDIR)"
	@cp bin/* "$(DESTDIR)/"

contrib:
	mkdir contrib

contrib/rclone-1.50.2.deb: contrib
	curl -sSfLo contrib/rclone-1.50.2.deb 'https://downloads.rclone.org/v1.50.2/rclone-v1.50.2-linux-amd64.deb'

docker: contrib/rclone-1.50.2.deb
	@echo "Building Docker image for v$(VERSION)"
	@docker build -t $(REGISTRY)ghetzel/clitools:$(VERSION) .
	docker tag $(REGISTRY)ghetzel/clitools:$(VERSION) $(REGISTRY)ghetzel/clitools:latest
	docker push $(REGISTRY)ghetzel/clitools:$(VERSION)
	docker push $(REGISTRY)ghetzel/clitools:latest


caspar-seq:
	casparctl -c 'stop 1-100'
	casparctl -c 'clear 1-100'
	casparctl play --wait "http://files.home/videos/other/LCARS/Torpedo%20Modifications.avi" -- AUTO
	casparctl play --wait "http://files.home/videos/other/LCARS/Emergency%20Operational%20Status.avi" -- LOOP 2 AUTO
	casparctl play --wait "http://files.home/videos/other/LCARS/Cardassian%20Federation%20Boundary.avi" -- AUTO
	casparctl play "http://files.home/videos/other/LCARS/Local%20Memory%20Allocation.avi" -- LOOP AUTO

.PHONY: $(TOOLS)