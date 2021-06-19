FROM golang:1.16-alpine3.13 AS firefly-builder
RUN apk add make gcc build-base curl
WORKDIR /firefly
ADD . .
ENV UI_RELEASE "https://github.com/hyperledger-labs/firefly-ui/releases/download/v0.1.0/v0.1.0_e41748c.tgz"
RUN mkdir /firefly/frontend \
 && curl -sLo - $UI_RELEASE | tar -C /firefly/frontend -zxvf -
RUN make
WORKDIR /firefly/solidity_firefly

FROM node:14-alpine3.11 AS solidity-builder
WORKDIR /firefly/solidity_firefly
ADD solidity_firefly .
RUN npm install
RUN npm config set user 0
RUN npx truffle compile

FROM alpine:latest  
WORKDIR /firefly
COPY --from=firefly-builder /firefly/firefly ./firefly
COPY --from=firefly-builder /firefly/frontend/ /firefly/frontend/
COPY --from=firefly-builder /firefly/db ./db
COPY --from=solidity-builder /firefly/solidity_firefly/build/contracts ./contracts
RUN ln -s /firefly/firefly /usr/bin/firefly
ENTRYPOINT [ "firefly" ]
