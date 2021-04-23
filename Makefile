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
		find . -type d | xargs -L 1 $(shell go list -f '{{.Target}}' golang.org/x/lint/golint) -set_exit_status
firefly: ${GOFILES}
		$(VGO) build -o ${BINARY_NAME} -ldflags "-X main.buildDate=`date -u +\"%Y-%m-%dT%H:%M:%SZ\"` -X main.buildVersion=$(BUILD_VERSION)" -tags=prod -tags=prod -v
build: ${BINARY_NAME}
clean: 
		$(VGO) clean
		rm -f *.so ${BINARY_NAME}
builddeps:
		$(VGO) get golang.org/x/lint/golint
deps: builddeps
		$(VGO) get
