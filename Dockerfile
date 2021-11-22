FROM golang:1.16-alpine3.13 AS firefly-builder
RUN apk add make gcc build-base curl git
WORKDIR /firefly
ADD go.mod go.sum ./
RUN go mod download
ENV UI_RELEASE "https://github.com/hyperledger/firefly-ui/releases/download/v0.4.1/v0.4.1_f09c392.tgz"
RUN mkdir /firefly/frontend \
 && curl -sLo - $UI_RELEASE | tar -C /firefly/frontend -zxvf -
ADD . .
RUN make build

FROM golang:1.16-alpine3.13 AS fabric-builder
RUN apk add  libc6-compat
WORKDIR /firefly/smart_contracts/fabric/firefly-go
ADD smart_contracts/fabric/firefly-go .
RUN GO111MODULE=on go mod vendor
WORKDIR /tmp/fabric
RUN wget https://github.com/hyperledger/fabric/releases/download/v2.3.2/hyperledger-fabric-linux-amd64-2.3.2.tar.gz
RUN tar -zxf hyperledger-fabric-linux-amd64-2.3.2.tar.gz
RUN touch core.yaml
RUN ./bin/peer lifecycle chaincode package /firefly/smart_contracts/fabric/firefly-go/firefly_fabric.tar.gz --path /firefly/smart_contracts/fabric/firefly-go --lang golang --label firefly_1.0

FROM node:14-alpine3.11 AS solidity-builder
WORKDIR /firefly/solidity_firefly
ADD smart_contracts/ethereum/solidity_firefly/package*.json .
RUN npm install
RUN npm config set user 0
ADD smart_contracts/ethereum/solidity_firefly .
RUN npx truffle compile

FROM alpine:latest
WORKDIR /firefly
COPY --from=firefly-builder /firefly/firefly ./firefly
COPY --from=firefly-builder /firefly/frontend/ /firefly/frontend/
COPY --from=firefly-builder /firefly/db ./db
COPY --from=solidity-builder /firefly/solidity_firefly/build/contracts ./contracts
COPY --from=fabric-builder /firefly/smart_contracts/fabric/firefly-go/firefly_fabric.tar.gz ./contracts/firefly_fabric.tar.gz
RUN ln -s /firefly/firefly /usr/bin/firefly
ENTRYPOINT [ "firefly" ]
