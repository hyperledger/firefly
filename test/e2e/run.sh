#!/bin/bash -x
set -euo pipefail

CWD=$(dirname "$0")
CLI=ff
STACK_DIR=~/.firefly/stacks
STACK_NAME=firefly-e2e
STACK_FILE=$STACK_DIR/$STACK_NAME/stack.json
DOWNLOAD_CLI=true
CREATE_STACK=true

if $DOWNLOAD_CLI
then
	GO111MODULE=off go get github.com/kaleido-io/firefly-cli/ff
fi

if $CREATE_STACK
then
	$CLI remove -f $STACK_NAME || true
	$CLI init $STACK_NAME 2
	$CLI start $STACK_NAME
	# Stack doesn't come up cleanly the first time...
	# TODO: fix this
	$CLI stop $STACK_NAME
	$CLI start $STACK_NAME
fi

export STACK_FILE
cd $CWD && go test -v .
