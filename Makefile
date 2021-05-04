VGO=go
BINARY_NAME=firefly
GOFILES := $(shell find cmd internal pkg -name '*.go' -print)
MOCKERY=mockery
.DELETE_ON_ERROR:

all: build test
test: deps lint
		$(VGO) test  ./... -cover -coverprofile=coverage.txt -covermode=atomic
coverage.html:
		$(VGO) tool cover -html=coverage.txt
coverage: test coverage.html
lint:
		$(shell go list -f '{{.Target}}' github.com/golangci/golangci-lint/cmd/golangci-lint) run
mocks: ${GOFILES}
		$(MOCKERY) --case underscore --dir internal/blockchain   --name Plugin       --output mocks/blockchainmocks  --outpkg blockchainmocks
		$(MOCKERY) --case underscore --dir internal/blockchain   --name Events       --output mocks/blockchainmocks  --outpkg blockchainmocks
		$(MOCKERY) --case underscore --dir internal/persistence  --name Plugin       --output mocks/persistencemocks --outpkg persistencemocks
		$(MOCKERY) --case underscore --dir internal/persistence  --name Events       --output mocks/persistencemocks --outpkg persistencemocks
		$(MOCKERY) --case underscore --dir internal/p2pfs        --name Plugin       --output mocks/p2pfsmocks       --outpkg p2pfsmocks
		$(MOCKERY) --case underscore --dir internal/p2pfs        --name Events       --output mocks/p2pfsmocks       --outpkg p2pfsmocks
		$(MOCKERY) --case underscore --dir internal/batching     --name BatchManager --output mocks/batchingmocks    --outpkg batchingmocks
		$(MOCKERY) --case underscore --dir internal/engine       --name Engine       --output mocks/enginemocks      --outpkg enginemocks
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
