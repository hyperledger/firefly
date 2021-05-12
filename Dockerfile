FROM golang:1.16.4-alpine3.13
WORKDIR /firefly
ADD . .
RUN apk add make gcc build-base
RUN make

ENTRYPOINT [ "firefly" ]