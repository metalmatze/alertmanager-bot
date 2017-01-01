EXECUTABLE ?= alertmanager-telegram
IMAGE ?= metalmatze/$(EXECUTABLE)
CI_BUILD_NUMBER ?= 0

LDFLAGS = -X main.BuildTime=$(shell date +%FT%T%z) -X main.Commit=$(shell git rev-parse --short=8 HEAD)
PACKAGES = $(shell go list ./... | grep -v /vendor/)

.PHONY: all
all: build

.PHONY: clean
clean:
	go clean -i ./...

$(EXECUTABLE): $(wildcard *.go)
	go build -ldflags '-w $(LDFLAGS)'

.PHONY: build
build: $(EXECUTABLE)

.PHONY: test
test:
	@for PKG in $(PACKAGES); do go test -cover -coverprofile $$GOPATH/src/$$PKG/coverage.out $$PKG || exit 1; done;

.PHONY: fmt
fmt:
	go fmt $(PACKAGES)

.PHONY: vet
vet:
	go vet $(PACKAGES)

.PHONY: lint
lint:
	@which golint > /dev/null; if [ $$? -ne 0 ]; then \
		go get -u github.com/golang/lint/golint; \
	fi
	for PKG in $(PACKAGES); do golint -set_exit_status $$PKG || exit 1; done;