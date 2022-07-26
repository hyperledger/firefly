ARG FIREFLY_BUILDER_TAG
ARG FABRIC_BUILDER_TAG
ARG FABRIC_BUILDER_PLATFORM
ARG SOLIDITY_BUILDER_TAG
ARG BASE_TAG
ARG BUILD_VERSION
ARG GIT_REF

FROM $FIREFLY_BUILDER_TAG AS firefly-builder
RUN apk add make gcc build-base curl git
WORKDIR /firefly
ADD go.mod go.sum ./
RUN go mod download
ADD . .
RUN make build BUILD_VERSION=$BUILD_VERSION GIT_REF=$GIT_REF

FROM --platform=$FABRIC_BUILDER_PLATFORM $FABRIC_BUILDER_TAG AS fabric-builder
RUN apk add libc6-compat
WORKDIR /firefly/smart_contracts/fabric/firefly-go
ADD smart_contracts/fabric/firefly-go .
RUN GO111MODULE=on go mod vendor
WORKDIR /tmp/fabric
RUN wget https://github.com/hyperledger/fabric/releases/download/v2.3.2/hyperledger-fabric-linux-amd64-2.3.2.tar.gz
RUN tar -zxf hyperledger-fabric-linux-amd64-2.3.2.tar.gz
RUN touch core.yaml
RUN ./bin/peer lifecycle chaincode package /firefly/smart_contracts/fabric/firefly-go/firefly_fabric.tar.gz --path /firefly/smart_contracts/fabric/firefly-go --lang golang --label firefly_1.0

FROM $SOLIDITY_BUILDER_TAG AS solidity-builder
WORKDIR /firefly/solidity_firefly
ADD smart_contracts/ethereum/solidity_firefly/ .
RUN apk add jq \
 && mkdir -p build/contracts \
 && cd contracts \
 && solc --combined-json abi,bin,devdoc -o ../build/contracts Firefly.sol \
 && cd ../build/contracts \
 && mv combined.json Firefly.json

FROM $BASE_TAG
ARG UI_TAG
ARG UI_RELEASE
RUN apk add --update --no-cache sqlite postgresql-client curl jq
WORKDIR /firefly
COPY --from=firefly-builder /firefly/firefly ./firefly
COPY --from=firefly-builder /firefly/db ./db
COPY --from=solidity-builder /firefly/solidity_firefly/build/contracts ./contracts
COPY --from=fabric-builder /firefly/smart_contracts/fabric/firefly-go/firefly_fabric.tar.gz ./contracts/firefly_fabric.tar.gz
ENV UI_RELEASE https://github.com/hyperledger/firefly-ui/releases/download/$UI_TAG/$UI_RELEASE.tgz
RUN mkdir /firefly/frontend \
 && curl -sLo - $UI_RELEASE | tar -C /firefly/frontend -zxvf -
RUN ln -s /firefly/firefly /usr/bin/firefly \
    && curl -sL "https://github.com/golang-migrate/migrate/releases/download/$(curl -sL https://api.github.com/repos/golang-migrate/migrate/releases/latest | jq -r '.name')/migrate.linux-amd64.tar.gz" | tar xz \
    && chmod +x ./migrate \
    && mv ./migrate /usr/bin/migrate
ENTRYPOINT [ "firefly" ]
