EXECUTABLE ?= alertmanager-bot
IMAGE ?= metalmatze/$(EXECUTABLE)
GO := CGO_ENABLED=0 go
DATE := $(shell date -u '+%FT%T%z')

LDFLAGS += -X main.Version=$(DRONE_TAG)
LDFLAGS += -X main.Revision=$(DRONE_COMMIT)
LDFLAGS += -X "main.BuildDate=$(DATE)"
LDFLAGS += -extldflags '-static'

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
		$(GO) get -u golang.org/x/lint/golint; \
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

README.md: deployments/examples/docker-compose.yaml deployments/examples/kubernetes.yaml
	embedmd -w README.md

deployments/examples/kubernetes.yaml: deployments/examples/kubernetes.jsonnet deployments/examples/values.jsonnet deployments/kubernetes.libsonnet
	jsonnetfmt -i deployments/kubernetes.libsonnet
	jsonnetfmt -i deployments/examples/kubernetes.jsonnet
	jsonnet deployments/examples/kubernetes.jsonnet | gojsontoyaml > deployments/examples/kubernetes.yaml

deployments/examples/docker-compose.yaml: deployments/examples/docker-compose.jsonnet deployments/examples/values.jsonnet deployments/docker-compose.libsonnet
	jsonnetfmt -i deployments/docker-compose.libsonnet
	jsonnetfmt -i deployments/examples/docker-compose.jsonnet
	jsonnet deployments/examples/docker-compose.jsonnet | gojsontoyaml > deployments/examples/docker-compose.yaml
