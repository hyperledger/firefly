#!/bin/bash -x

set -o pipefail

 if [[ ! -x `which jq` ]]; then echo "Please install \"jq\" to continue"; exit 1; fi

CWD=$(dirname "$0")
CLI="ff -v --ansi never"
CLI_VERSION=$(cat $CWD/../../manifest.json | jq -r .cli.tag)
STACKS_DIR=~/.firefly/stacks

create_accounts() {
  if [ "$TEST_SUITE" == "TestEthereumMultipartyE2ESuite" ]; then
      # Create 4 new accounts for use in testing
      for i in {1..4}
      do
          $CLI accounts create $STACK_NAME
      done
  elif [ "$TEST_SUITE" == "TestFabricMultipartyE2ESuite" ]; then
      # Create 4 new accounts for the first org for use in testing
      for i in {1..3}
      do
          $CLI accounts create $STACK_NAME org_0 user_$(openssl rand -hex 10)
      done
      # Create one account for the second org
      $CLI accounts create $STACK_NAME org_1 user_$(openssl rand -hex 10)
  fi
}

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

if [ -z "${MULTIPARTY_ENABLED}" ]; then
  MULTIPARTY_ENABLED=true
fi

if [ -z "${STACK_FILE}" ]; then
  STACK_FILE=$STACKS_DIR/$STACK_NAME/stack.json
fi

if [ -z "${STACK_STATE}" ]; then
  STACK_STATE=$STACKS_DIR/$STACK_NAME/runtime/stackState.json
fi

if [ -z "${BLOCKCHAIN_PROVIDER}" ]; then
  BLOCKCHAIN_PROVIDER=geth
fi

if [ -z "${TOKENS_PROVIDER}" ]; then
  TOKENS_PROVIDER=erc20_erc721
fi

if [ -z "${TEST_SUITE}" ]; then
  if [ "${BLOCKCHAIN_PROVIDER}" == "fabric" ]; then
    if [ "${MULTIPARTY_ENABLED}" == "true" ]; then
      TEST_SUITE=TestFabricMultipartyE2ESuite
    else
      TEST_SUITE=TestFabricGatewayE2ESuite
    fi
  else
    if [ "${MULTIPARTY_ENABLED}" == "true" ]; then
      TEST_SUITE=TestEthereumMultipartyE2ESuite
    else
      TEST_SUITE=TestEthereumGatewayE2ESuite
    fi
  fi
fi

cd $CWD

if [ "$CREATE_STACK" == "true" ]; then
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
  $CLI init --prometheus-enabled --database $DATABASE_TYPE $STACK_NAME 2 --blockchain-provider $BLOCKCHAIN_PROVIDER --token-providers $TOKENS_PROVIDER --manifest ../../manifest.json $EXTRA_INIT_ARGS --sandbox-enabled=false --multiparty=$MULTIPARTY_ENABLED
  checkOk $?

  $CLI pull $STACK_NAME -r 3
  checkOk $?

  $CLI start -b $STACK_NAME
  checkOk $?
fi

create_accounts

$CLI info $STACK_NAME
checkOk $?

export STACK_FILE
export STACK_STATE

runTest() {
  go clean -testcache && go test -v -p 1 ./multiparty ./gateway -run $TEST_SUITE
  checkOk $?
}
runTest

if [ "$RESTART" == "true" ]; then
  $CLI stop $STACK_NAME
  checkOk $?

  $CLI start $STACK_NAME
  checkOk $?

  create_accounts
  runTest
fi
