FROM golang:1.18

WORKDIR /app
COPY  firefly.go go.mod go.sum ./
COPY chaincode/ ./chaincode/
RUN ls -la ./ \
    && GO111MODULE=on GOOS=linux CGO_ENABLED=0 go build -o firefly.bin firefly.go

