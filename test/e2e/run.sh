#!/bin/bash -x
set -eo pipefail

CWD=$(dirname "$0")
CLI="ff -v --ansi never"
STACK_DIR=~/.firefly/stacks
STACK_NAME=firefly-e2e
STACK_FILE=$STACK_DIR/$STACK_NAME/stack.json

if [ -z "${DOWNLOAD_CLI}" ]; then
  DOWNLOAD_CLI=true
fi

if [ -z "${CREATE_STACK}" ]; then
  CREATE_STACK=true
fi

if [ -z "${BUILD_FIREFLY}" ]; then
  BUILD_FIREFLY=true
fi

cd $CWD

if [ "$BUILD_FIREFLY" == "true" ]
then
	docker build -t ghcr.io/hyperledger-labs/firefly:latest ../..
fi

if [ "$DOWNLOAD_CLI" == "true" ]
then
	go install github.com/hyperledger-labs/firefly-cli/ff@latest
fi

if [ "$CREATE_STACK" == "true" ]
then
	$CLI remove -f $STACK_NAME || true
	$CLI init $STACK_NAME 2
	$CLI start -n $STACK_NAME
fi

export STACK_FILE
go clean -testcache && go test -v .
