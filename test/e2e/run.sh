#!/bin/bash -x

set -o pipefail

 if [[ ! -x `which jq` ]]; then echo "Please install \"jq\" to continue"; exit 1; fi

CWD=$(dirname "$0")
CLI="ff -v --ansi never"
CLI_VERSION=$(cat $CWD/../../manifest.json | jq -r .cli.tag)
STACK_DIR=~/.firefly/stacks

checkOk() {
  local rc=$1

  WORKDIR=${GITHUB_WORKSPACE}
  if [ -z "$WORKDIR" ]; then WORKDIR=.; fi

  mkdir -p "${WORKDIR}/containerlogs"
  $CLI logs $STACK_NAME > "${WORKDIR}/containerlogs/logs.txt"
  if [ $rc -eq 0 ]; then rc=$?; fi

  if [ $rc -ne 0 ]; then exit $rc; fi
}

if [ -z "${STACK_NAME}" ]; then
  STACK_NAME=firefly_e2e
fi

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


if [ -z "${BLOCKCHAIN_PROVIDER}" ]; then
  BLOCKCHAIN_PROVIDER=geth
fi

if [ -z "${TOKENS_PROVIDER}" ]; then
  TOKENS_PROVIDER=erc1155
fi

if [ -z "${TEST_SUITE}" ]; then
  TEST_SUITE=TestEthereumE2ESuite
fi

cd $CWD

if [ "$CREATE_STACK" == "true" ]; then
  $CLI remove -f firefly-e2e  # TODO: remove
  $CLI remove -f $STACK_NAME
fi

if [ "$BUILD_FIREFLY" == "true" ]; then
  make -C ../.. docker
  checkOk $?
fi

if [ "$DOWNLOAD_CLI" == "true" ]; then
  go install github.com/hyperledger/firefly-cli/ff@$CLI_VERSION
  checkOk $?
fi

if [ "$CREATE_STACK" == "true" ]; then
  $CLI init --prometheus-enabled --database $DATABASE_TYPE $STACK_NAME 2 --blockchain-provider $BLOCKCHAIN_PROVIDER --token-providers $TOKENS_PROVIDER --manifest ../../manifest.json
  checkOk $?

  $CLI pull $STACK_NAME -r 3
  checkOk $?

  $CLI start -b $STACK_NAME
  checkOk $?

  if [ "$TEST_SUITE" == "TestEthereumE2ESuite" ]; then
      prefix='contract address: '
      output=$($CLI deploy $STACK_NAME ../data/simplestorage/simple_storage.json | grep address)
      CONTRACT_ADDRESS=${output#"$prefix"}
      export CONTRACT_ADDRESS
  fi
fi

$CLI info $STACK_NAME
checkOk $?

export STACK_FILE

go clean -testcache && go test -v . -run $TEST_SUITE
checkOk $?

if [ "$RESTART" == "true" ]; then
  $CLI stop $STACK_NAME
  checkOk $?

  $CLI start $STACK_NAME
  checkOk $?

  go clean -testcache && go test -v . -run $TEST_SUITE
  checkOk $?
fi
