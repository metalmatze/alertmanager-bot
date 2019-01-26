EXECUTABLE ?= alertmanager-bot
IMAGE ?= metalmatze/$(EXECUTABLE)
GO := CGO_ENABLED=0 go
DATE := $(shell date -u '+%FT%T%z')

LDFLAGS += -X main.Version=$(DRONE_TAG)
LDFLAGS += -X main.Revision=$(DRONE_COMMIT)
LDFLAGS += -X "main.BuildDate=$(DATE)"
LDFLAGS += -extldflags '-static'

SWAGGER = docker run \
	--user=$(shell id -u $(USER)):$(shell id -g $(USER)) \
	--rm \
	-v $(shell pwd):/go/src/github.com/prometheus/alertmanager \
	-w /go/src/github.com/prometheus/alertmanager quay.io/goswagger/swagger:v0.18.0

PACKAGES = $(shell go list ./... | grep -v /vendor/)

.PHONY: all
all: build

.PHONY: clean
clean:
	$(GO) clean -i ./...
	rm -rf dist/

.PHONY: fmt
fmt:
	$(GO) fmt $(PACKAGES)

.PHONY: vet
vet:
	$(GO) vet $(PACKAGES)

.PHONY: lint
lint:
	@which golint > /dev/null; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/golang/lint/golint; \
	fi
	for PKG in $(PACKAGES); do golint -set_exit_status $$PKG || exit 1; done;

.PHONY: test
test:
	@for PKG in $(PACKAGES); do $(GO) test -cover -coverprofile $$GOPATH/src/$$PKG/coverage.out $$PKG || exit 1; done;

.PHONY: build
build:
	$(GO) build -v -ldflags '-w $(LDFLAGS)' ./cmd/alertmanager-bot

.PHONY: release
release:
	@which gox > /dev/null; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/mitchellh/gox; \
	fi
	CGO_ENABLED=0 gox -arch="386 amd64 arm" -verbose -ldflags '-w $(LDFLAGS)' -output="dist/$(EXECUTABLE)-${DRONE_TAG}-{{.OS}}-{{.Arch}}" ./cmd/alertmanager-bot/

.PHONY: apiv2
apiv2: pkg/alertmanager/api/v2/openapi.yaml
	-rm -rf pkg/alertmanager/api/v2/{client,models}
	$(SWAGGER) generate client -f pkg/alertmanager/api/v2/openapi.yaml -A alertmanager --target pkg/alertmanager/api/v2/
