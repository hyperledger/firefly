FROM golang:1.16-alpine3.13 AS firefly-builder
RUN apk add make gcc build-base
WORKDIR /firefly
ADD . .
RUN make
WORKDIR /firefly/solidity_firefly

FROM node:14-alpine3.11 AS solidity-builder
WORKDIR /firefly/solidity_firefly
ADD solidity_firefly .
RUN npm install
RUN npm config set user 0
RUN npx truffle compile

FROM node:14-alpine3.11 AS firefly-ui-builder
RUN apk add git
RUN git clone https://github.com/hyperledger-labs/firefly-ui.git
WORKDIR /firefly-ui
RUN npm install
RUN PUBLIC_URL="/ui" npm run build

FROM alpine:latest  
WORKDIR /firefly
COPY --from=firefly-builder /firefly/firefly ./firefly
COPY --from=firefly-builder /firefly/db ./db
COPY --from=solidity-builder /firefly/solidity_firefly/build/contracts ./contracts
COPY --from=firefly-ui-builder /firefly-ui/build ./frontend
RUN ln -s /firefly/firefly /usr/bin/firefly
ENTRYPOINT [ "firefly" ]
