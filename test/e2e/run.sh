#!/bin/bash -x

set -o pipefail

CWD=$(dirname "$0")
CLI="ff -v --ansi never"
STACK_DIR=~/.firefly/stacks
STACK_NAME=firefly-e2e
RC=0

if [ -z "${DOWNLOAD_CLI}" ]; then
  DOWNLOAD_CLI=true
fi

if [ -z "${CREATE_STACK}" ]; then
  CREATE_STACK=true
fi

if [ -z "${BUILD_FIREFLY}" ]; then
  BUILD_FIREFLY=true
fi

if [ -z "${DATABASE_TYPE}" ]; then
  # Can also set to "postgres"
  DATABASE_TYPE=sqlite3
fi

if [ -z "${STACK_FILE}" ]; then
  STACK_FILE=$STACK_DIR/$STACK_NAME/stack.json
fi

cd $CWD

if [ $RC -eq 0 ] && [ "$CREATE_STACK" == "true" ]; then
  $CLI remove -f $STACK_NAME || true
  RC=$?
fi

if [ $RC -eq 0 ] && [ "$BUILD_FIREFLY" == "true" ]; then
  docker build -t ghcr.io/hyperledger/firefly:latest ../..
  RC=$?
fi

if [ $RC -eq 0 ] && [ "$DOWNLOAD_CLI" == "true" ]; then
  go install github.com/hyperledger/firefly-cli/ff@latest
  RC=$?
fi

if [ "$CREATE_STACK" == "true" ]; then
  if [ $RC -eq 0 ]; then
    $CLI init --database $DATABASE_TYPE $STACK_NAME 2
    RC=$?
  fi

  if [ $RC -eq 0 ]; then
    $CLI start -nb $STACK_NAME
    RC=$?
  fi
fi

if [ $RC -eq 0 ]; then
  $CLI info $STACK_NAME
  RC=$?
fi

export STACK_FILE

if [ $RC -eq 0 ]; then
  go clean -testcache && go test -v .
  RC=$?
fi

WORKDIR=${GITHUB_WORKSPACE}
if [ -z "$WORKDIR" ]; then WORKDIR=$CWD; fi

mkdir -p "${WORKDIR}/containerlogs"
$CLI logs $STACK_NAME > "${WORKDIR}/containerlogs/logs.txt"
if [ $RC -eq 0 ]; then RC=$?; fi

exit $RC
