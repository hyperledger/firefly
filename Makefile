VGO=go
BINARY_NAME=firefly
GOFILES := $(shell find cmd internal pkg -name '*.go' -print)
GOBIN := $(shell $(VGO) env GOPATH)/bin
LINT := $(GOBIN)/golangci-lint
MOCKERY := $(GOBIN)/mockery

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
lint: ${LINT}
		GOGC=20 $(LINT) run -v --timeout 5m
${MOCKERY}:
		$(VGO) install github.com/vektra/mockery/cmd/mockery@latest
${LINT}:
		$(VGO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest


define makemock
mocks: mocks-$(strip $(1))-$(strip $(2))
mocks-$(strip $(1))-$(strip $(2)): ${MOCKERY}
	${MOCKERY} --case underscore --dir $(1) --name $(2) --outpkg $(3) --output mocks/$(strip $(3))
endef

$(eval $(call makemock, pkg/blockchain,            Plugin,             blockchainmocks))
$(eval $(call makemock, pkg/blockchain,            Callbacks,          blockchainmocks))
$(eval $(call makemock, pkg/database,              Plugin,             databasemocks))
$(eval $(call makemock, pkg/database,              Callbacks,          databasemocks))
$(eval $(call makemock, pkg/publicstorage,         Plugin,             publicstoragemocks))
$(eval $(call makemock, pkg/publicstorage,         Callbacks,          publicstoragemocks))
$(eval $(call makemock, pkg/events,                Plugin,             eventsmocks))
$(eval $(call makemock, pkg/events,                PluginAll,          eventsmocks))
$(eval $(call makemock, pkg/events,                Callbacks,          eventsmocks))
$(eval $(call makemock, pkg/identity,              Plugin,             identitymocks))
$(eval $(call makemock, pkg/identity,              Callbacks,          identitymocks))
$(eval $(call makemock, pkg/dataexchange,          Plugin,             dataexchangemocks))
$(eval $(call makemock, pkg/dataexchange,          Callbacks,          dataexchangemocks))
$(eval $(call makemock, pkg/tokens,                Plugin,             tokenmocks))
$(eval $(call makemock, pkg/tokens,                Callbacks,          tokenmocks))
$(eval $(call makemock, pkg/wsclient,              WSClient,           wsmocks))
$(eval $(call makemock, internal/txcommon,         Helper,             txcommonmocks))
$(eval $(call makemock, internal/identity,         Manager,            identitymanagermocks))
$(eval $(call makemock, internal/batchpin,         Submitter,          batchpinmocks))
$(eval $(call makemock, internal/sysmessaging,     SystemEvents,       sysmessagingmocks))
$(eval $(call makemock, internal/sysmessaging,     MessageSender,      sysmessagingmocks))
$(eval $(call makemock, internal/sysmessaging,     LocalNodeInfo,      sysmessagingmocks))
$(eval $(call makemock, internal/syncasync,        Bridge,             syncasyncmocks))
$(eval $(call makemock, internal/data,             Manager,            datamocks))
$(eval $(call makemock, internal/batch,            Manager,            batchmocks))
$(eval $(call makemock, internal/broadcast,        Manager,            broadcastmocks))
$(eval $(call makemock, internal/privatemessaging, Manager,            privatemessagingmocks))
$(eval $(call makemock, internal/definitions,      DefinitionHandlers, definitionsmocks))
$(eval $(call makemock, internal/events,           EventManager,       eventmocks))
$(eval $(call makemock, internal/networkmap,       Manager,            networkmapmocks))
$(eval $(call makemock, internal/assets,           Manager,            assetmocks))
$(eval $(call makemock, internal/contracts,        Manager,            contractmocks))
$(eval $(call makemock, internal/oapiffi,          FFISwaggerGen,      oapiffimocks))
$(eval $(call makemock, internal/orchestrator,     Orchestrator,       orchestratormocks))
$(eval $(call makemock, internal/apiserver,        Server,             apiservermocks))
$(eval $(call makemock, internal/apiserver,        IServer,            apiservermocks))
$(eval $(call makemock, internal/metrics,          Manager,            metricsmocks))
$(eval $(call makemock, internal/operations,       Manager,            operationmocks))

firefly-nocgo: ${GOFILES}
		CGO_ENABLED=0 $(VGO) build -o ${BINARY_NAME}-nocgo -ldflags "-X main.buildDate=`date -u +\"%Y-%m-%dT%H:%M:%SZ\"` -X main.buildVersion=$(BUILD_VERSION)" -tags=prod -tags=prod -v
firefly: ${GOFILES}
		$(VGO) build -o ${BINARY_NAME} -ldflags "-X main.buildDate=`date -u +\"%Y-%m-%dT%H:%M:%SZ\"` -X main.buildVersion=$(BUILD_VERSION)" -tags=prod -tags=prod -v
go-mod-tidy: .ALWAYS
		$(VGO) mod tidy
build: firefly-nocgo firefly
e2e: build
		./test/e2e/run.sh
.ALWAYS: ;
e2e-rebuild: .ALWAYS
		DOWNLOAD_CLI=false BUILD_FIREFLY=false CREATE_STACK=false ./test/e2e/run.sh
clean:
		$(VGO) clean
		rm -f *.so ${BINARY_NAME}
deps:
		$(VGO) get
swagger:
		$(VGO) test ./internal/apiserver -timeout=10s -tags swagger
manifest:
		./manifestgen.sh
docker:
		./docker_build.sh $(DOCKER_ARGS)
