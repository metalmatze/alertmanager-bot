EXECUTABLE ?= alertmanager-telegram
IMAGE ?= metalmatze/$(EXECUTABLE)
CI_BUILD_NUMBER ?= 0

LDFLAGS = -X "main.buildDate=$(shell date -u '+%Y-%m-%d %H:%M:%S %Z')"
PACKAGES = $(shell go list ./... | grep -v /vendor/)

.PHONY: all
all: install

.PHONY: clean
clean:
	go clean -i ./...

.PHONY: deps
deps:
	go get -u github.com/govend/govend
	govend -v

.PHONY: fmt
fmt:
	go fmt $(PACKAGES)

.PHONY: vet
vet:
	go vet $(PACKAGES)

.PHONY: test
test:
	@for PKG in $(PACKAGES); do go test -cover -coverprofile $$GOPATH/src/$$PKG/coverage.out $$PKG || exit 1; done;

$(EXECUTABLE): $(wildcard *.go)
	go install -ldflags '-s -w $(LDFLAGS)'

.PHONY: install
install: $(EXECUTABLE)
