#!/bin/bash -x

set -o pipefail

CWD=$(dirname "$0")
CLI="ff -v --ansi never"
STACK_DIR=~/.firefly/stacks
STACK_NAME=firefly-e2e

checkOk() {
  local rc=$1

  WORKDIR=${GITHUB_WORKSPACE}
  if [ -z "$WORKDIR" ]; then WORKDIR=.; fi

  mkdir -p "${WORKDIR}/containerlogs"
  $CLI logs $STACK_NAME > "${WORKDIR}/containerlogs/logs.txt"
  if [ $rc -eq 0 ]; then rc=$?; fi

  if [ $rc -ne 0 ]; then exit $rc; fi
}

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

if [ "$CREATE_STACK" == "true" ]; then
  $CLI remove -f $STACK_NAME || true
  checkOk $?
fi

if [ "$BUILD_FIREFLY" == "true" ]; then
  docker build -t ghcr.io/hyperledger/firefly:latest ../..
  checkOk $?
fi

if [ "$DOWNLOAD_CLI" == "true" ]; then
  go install github.com/hyperledger/firefly-cli/ff@latest
  checkOk $?
fi

if [ "$CREATE_STACK" == "true" ]; then
  $CLI init --database $DATABASE_TYPE $STACK_NAME 2
  checkOk $?

  $CLI start -nb $STACK_NAME
  checkOk $?
fi

$CLI info $STACK_NAME
checkOk $?

export STACK_FILE

go clean -testcache && go test -v .
checkOk $?
