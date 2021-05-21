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
		$(shell go list -f '{{.Target}}' github.com/golangci/golangci-lint/cmd/golangci-lint) run -E gofmt -E gocritic
mocks: ${GOFILES}
		$(MOCKERY) --case underscore --dir pkg/blockchain        --name Plugin           --output mocks/blockchainmocks     --outpkg blockchainmocks
		$(MOCKERY) --case underscore --dir pkg/blockchain        --name Callbacks        --output mocks/blockchainmocks     --outpkg blockchainmocks
		$(MOCKERY) --case underscore --dir pkg/database          --name Plugin           --output mocks/databasemocks       --outpkg databasemocks
		$(MOCKERY) --case underscore --dir pkg/database          --name Callbacks        --output mocks/databasemocks       --outpkg databasemocks
		$(MOCKERY) --case underscore --dir pkg/publicstorage     --name Plugin           --output mocks/publicstoragemocks  --outpkg publicstoragemocks
		$(MOCKERY) --case underscore --dir pkg/publicstorage     --name Callbacks        --output mocks/publicstoragemocks  --outpkg publicstoragemocks
		$(MOCKERY) --case underscore --dir pkg/events    				 --name Plugin           --output mocks/eventsmocks 			  --outpkg eventsmocks
		$(MOCKERY) --case underscore --dir pkg/events    				 --name Callbacks        --output mocks/eventsmocks 				--outpkg eventsmocks
		$(MOCKERY) --case underscore --dir internal/events       --name EventManager     --output mocks/eventmocks          --outpkg eventmocks
		$(MOCKERY) --case underscore --dir internal/batch        --name BatchManager     --output mocks/batchmocks          --outpkg batchmocks
		$(MOCKERY) --case underscore --dir internal/broadcast    --name BroadcastManager --output mocks/broadcastmocks      --outpkg broadcastmocks
		$(MOCKERY) --case underscore --dir internal/orchestrator --name Orchestrator     --output mocks/orchestratormocks   --outpkg orchestratormocks
		$(MOCKERY) --case underscore --dir internal/wsclient     --name WSClient         --output mocks/wsmocks             --outpkg wsmocks
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
