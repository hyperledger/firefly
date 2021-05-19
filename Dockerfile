FROM golang:1.16.4-alpine AS builder
WORKDIR /firefly
ADD . .
RUN apk add make gcc build-base nodejs npm python3
RUN make
WORKDIR /firefly/solidity_firefly
RUN npm install
RUN npm config set user 0
RUN npx truffle compile

FROM alpine:latest  
WORKDIR /firefly
COPY --from=builder /firefly/firefly ./firefly
COPY --from=builder /firefly/solidity_firefly/build/contracts ./contracts
COPY --from=builder /firefly/db ./db
RUN ln -s /firefly/firefly /usr/bin/firefly
ENTRYPOINT [ "firefly" ]