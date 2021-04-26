VGO=go
BINARY_NAME=firefly
GOFILES := $(shell find cmd internal pkg -name '*.go' -print)
.DELETE_ON_ERROR:

all: build test
test: deps lint
		$(VGO) test  ./... -cover -coverprofile=coverage.txt -covermode=atomic
coverage.html:
		$(VGO) tool cover -html=coverage.txt
coverage: test coverage.html
lint:
		$(shell go list -f '{{.Target}}' github.com/golangci/golangci-lint/cmd/golangci-lint) run
firefly: ${GOFILES}
		$(VGO) build -o ${BINARY_NAME} -ldflags "-X main.buildDate=`date -u +\"%Y-%m-%dT%H:%M:%SZ\"` -X main.buildVersion=$(BUILD_VERSION)" -tags=prod -tags=prod -v
build: ${BINARY_NAME}
clean: 
		$(VGO) clean
		rm -f *.so ${BINARY_NAME}
builddeps:
		$(VGO) get github.com/golangci/golangci-lint/cmd/golangci-lint
deps: builddeps
		$(VGO) get
