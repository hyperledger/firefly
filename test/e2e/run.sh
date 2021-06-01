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

if $BUILD_FIREFLY
then
	docker build -t kaleidoinc/firefly:latest .
fi

if $DOWNLOAD_CLI
then
	go install github.com/kaleido-io/firefly-cli/ff@latest
fi

if $CREATE_STACK
then
	$CLI remove -f $STACK_NAME || true
	$CLI init $STACK_NAME 2
	$CLI start $STACK_NAME
fi

sleep 5s
docker logs firefly-e2e_firefly_core_0_1
docker ps

export STACK_FILE
cd $CWD && go test -v .
