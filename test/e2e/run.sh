#!/bin/bash -x
set -euo pipefail

CWD=$(dirname "$0")
CLI="ff -v --ansi never"
STACK_DIR=~/.firefly/stacks
STACK_NAME=firefly-e2e
STACK_FILE=$STACK_DIR/$STACK_NAME/stack.json
DOWNLOAD_CLI=true
CREATE_STACK=true
BUILD_FIREFLY=true

cd $CWD

if $BUILD_FIREFLY
then
	docker build -t ghcr.io/hyperledger-labs/firefly:latest ../..
fi

if $DOWNLOAD_CLI
then
	go install github.com/hyperledger-labs/firefly-cli/ff@latest
fi

if $CREATE_STACK
then
	$CLI remove -f $STACK_NAME || true
	$CLI init $STACK_NAME 2
	$CLI start -n $STACK_NAME
fi

export STACK_FILE
go clean -testcache && go test -v .
