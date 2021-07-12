VGO=go
BINARY_NAME=firefly
GOFILES := $(shell find cmd internal pkg -name '*.go' -print)
# Expect that FireFly compiles with CGO disabled
CGO_ENABLED=0
GOGC=30
.DELETE_ON_ERROR:

all: build test go-mod-tidy
test: deps lint
		$(VGO) test ./internal/... ./pkg/... ./cmd/... -cover -coverprofile=coverage.txt -covermode=atomic -timeout=10s
coverage.html:
		$(VGO) tool cover -html=coverage.txt
coverage: test coverage.html
lint:
		GOGC=20 $(shell go list -f '{{.Target}}' github.com/golangci/golangci-lint/cmd/golangci-lint) run -v --timeout 5m
mockery: .ALWAYS
		go get github.com/vektra/mockery/cmd/mockery
mocks: mockery ${GOFILES}
		$(eval MOCKERY := $(shell go list -f '{{.Target}}' github.com/vektra/mockery/cmd/mockery))
		${MOCKERY} --case underscore --dir pkg/blockchain            --name Plugin           --output mocks/blockchainmocks       --outpkg blockchainmocks
		${MOCKERY} --case underscore --dir pkg/blockchain            --name Callbacks        --output mocks/blockchainmocks       --outpkg blockchainmocks
		${MOCKERY} --case underscore --dir pkg/database              --name Plugin           --output mocks/databasemocks         --outpkg databasemocks
		${MOCKERY} --case underscore --dir pkg/database              --name Callbacks        --output mocks/databasemocks         --outpkg databasemocks
		${MOCKERY} --case underscore --dir pkg/publicstorage         --name Plugin           --output mocks/publicstoragemocks    --outpkg publicstoragemocks
		${MOCKERY} --case underscore --dir pkg/publicstorage         --name Callbacks        --output mocks/publicstoragemocks    --outpkg publicstoragemocks
		${MOCKERY} --case underscore --dir pkg/events                --name Plugin           --output mocks/eventsmocks           --outpkg eventsmocks
		${MOCKERY} --case underscore --dir pkg/events                --name Callbacks        --output mocks/eventsmocks           --outpkg eventsmocks
		${MOCKERY} --case underscore --dir pkg/identity              --name Plugin           --output mocks/identitymocks         --outpkg identitymocks
		${MOCKERY} --case underscore --dir pkg/identity              --name Callbacks        --output mocks/identitymocks         --outpkg identitymocks
		${MOCKERY} --case underscore --dir pkg/dataexchange          --name Plugin           --output mocks/dataexchangemocks     --outpkg dataexchangemocks
		${MOCKERY} --case underscore --dir pkg/dataexchange          --name Callbacks        --output mocks/dataexchangemocks     --outpkg dataexchangemocks
		${MOCKERY} --case underscore --dir internal/data             --name Manager          --output mocks/datamocks             --outpkg datamocks
		${MOCKERY} --case underscore --dir internal/batch            --name Manager          --output mocks/batchmocks            --outpkg batchmocks
		${MOCKERY} --case underscore --dir internal/broadcast        --name Manager          --output mocks/broadcastmocks        --outpkg broadcastmocks
		${MOCKERY} --case underscore --dir internal/privatemessaging --name Manager          --output mocks/privatemessagingmocks --outpkg privatemessagingmocks
		${MOCKERY} --case underscore --dir internal/events           --name EventManager     --output mocks/eventmocks            --outpkg eventmocks
		${MOCKERY} --case underscore --dir internal/networkmap       --name Manager          --output mocks/networkmapmocks       --outpkg networkmapmocks
		${MOCKERY} --case underscore --dir internal/syncasync        --name Bridge           --output mocks/syncasyncmocks        --outpkg syncasyncmocks
		${MOCKERY} --case underscore --dir internal/wsclient         --name WSClient         --output mocks/wsmocks               --outpkg wsmocks
		${MOCKERY} --case underscore --dir internal/orchestrator     --name Orchestrator     --output mocks/orchestratormocks     --outpkg orchestratormocks
		${MOCKERY} --case underscore --dir internal/apiserver        --name Server           --output mocks/apiservermocks        --outpkg apiservermocks
firefly-nocgo: ${GOFILES}		
		CGO_ENABLED=0 $(VGO) build -o ${BINARY_NAME}-nocgo -ldflags "-X main.buildDate=`date -u +\"%Y-%m-%dT%H:%M:%SZ\"` -X main.buildVersion=$(BUILD_VERSION)" -tags=prod -tags=prod -v
firefly: ${GOFILES}
		$(VGO) build -o ${BINARY_NAME} -ldflags "-X main.buildDate=`date -u +\"%Y-%m-%dT%H:%M:%SZ\"` -X main.buildVersion=$(BUILD_VERSION)" -tags=prod -tags=prod -v
go-mod-tidy: .ALWAYS
		go mod tidy
build: firefly-nocgo firefly
e2e: build
		./test/e2e/run.sh
.ALWAYS: ;
e2e-rebuild: .ALWAYS
		DOWNLOAD_CLI=false BUILD_FIREFLY=false CREATE_STACK=false ./test/e2e/run.sh
clean: 
		$(VGO) clean
		rm -f *.so ${BINARY_NAME}
builddeps:
		$(VGO) get github.com/golangci/golangci-lint/cmd/golangci-lint
deps: builddeps
		$(VGO) get
swagger:
		$(VGO) test ./internal/apiserver -timeout=10s -tags swagger