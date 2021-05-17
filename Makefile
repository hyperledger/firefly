VGO=go
BINARY_NAME=firefly
GOFILES := $(shell find cmd internal pkg -name '*.go' -print)
MOCKERY=mockery
.DELETE_ON_ERROR:

all: build test
test: deps lint
		$(VGO) test ./internal/... ./pkg/... ./cmd/... -cover -coverprofile=coverage.txt -covermode=atomic
coverage.html:
		$(VGO) tool cover -html=coverage.txt
coverage: test coverage.html
lint:
		$(shell go list -f '{{.Target}}' github.com/golangci/golangci-lint/cmd/golangci-lint) run
mocks: ${GOFILES}
		$(MOCKERY) --case underscore --dir internal/blockchain        --name Plugin           --output mocks/blockchainmocks    --outpkg blockchainmocks
		$(MOCKERY) --case underscore --dir internal/blockchain        --name Events           --output mocks/blockchainmocks    --outpkg blockchainmocks
		$(MOCKERY) --case underscore --dir internal/database          --name Plugin           --output mocks/databasemocks      --outpkg databasemocks
		$(MOCKERY) --case underscore --dir internal/database          --name Events           --output mocks/databasemocks      --outpkg databasemocks
		$(MOCKERY) --case underscore --dir internal/publicstorage     --name Plugin           --output mocks/publicstoragemocks --outpkg publicstoragemocks
		$(MOCKERY) --case underscore --dir internal/publicstorage     --name Events           --output mocks/publicstoragemocks --outpkg publicstoragemocks
		$(MOCKERY) --case underscore --dir internal/events            --name EventManager     --output mocks/eventmocks         --outpkg eventmocks
		$(MOCKERY) --case underscore --dir internal/batching          --name BatchManager     --output mocks/batchingmocks      --outpkg batchingmocks
		$(MOCKERY) --case underscore --dir internal/broadcast         --name BroadcastManager --output mocks/broadcastmocks     --outpkg broadcastmocks
		$(MOCKERY) --case underscore --dir internal/orchestrator      --name Orchestrator     --output mocks/orchestratormocks  --outpkg orchestratormocks
		$(MOCKERY) --case underscore --dir internal/wsclient          --name WSClient         --output mocks/wsmocks            --outpkg wsmocks
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
