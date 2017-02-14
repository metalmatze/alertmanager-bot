EXECUTABLE ?= alertmanager-telegram
IMAGE ?= metalmatze/$(EXECUTABLE)
GO := CGO_ENABLED=0 go

LDFLAGS += -X main.BuildTime=$(shell date +%FT%T%z)
LDFLAGS += -X "main.Version=$(VERSION)"
LDFLAGS += -X main.Commit=$(shell git rev-parse --short=8 HEAD)
LDFLAGS += -extldflags '-static'

PACKAGES = $(shell go list ./... | grep -v /vendor/)

.PHONY: all
all: build

.PHONY: clean
clean:
	$(GO) clean -i ./...

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

.PHONY: errcheck
errcheck:
	@which errcheck > /dev/null; if [ $$? -ne 0 ]; then \
		$(GO) get -u github.com/kisielk/errcheck; \
	fi
	for PKG in $(PACKAGES); do errcheck $$PKG || exit 1; done;

.PHONY: test
test:
	@for PKG in $(PACKAGES); do go test -cover -coverprofile $$GOPATH/src/$$PKG/coverage.out $$PKG || exit 1; done;

$(EXECUTABLE): $(wildcard *.go)
	$(GO) build -v -ldflags '-w $(LDFLAGS)'

.PHONY: build
build: $(EXECUTABLE)

.PHONY: install
install:
	$(GO) install -v -ldflags '-w $(LDFLAGS)'
