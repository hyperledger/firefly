#!/bin/bash

if [ -z "$STACK_NAME" ]; then
  echo "Error: STACK_NAME must be set"
  exit 1
fi

if [ -z "$CHAINCODE_NAME" ]; then
  echo "Error: CHAINCODE_NAME must be set"
  exit 1
fi

TEST_DIR="$(cd "$(dirname $0)/.." && pwd)"
CHAINCODE="$TEST_DIR/data/assetcreator"
CHAINCODE_VERSION=1.0

CHANNEL=firefly
NETWORK=${STACK_NAME}_default
ORG_NAME=Org1MSP
ORG_DIR=/etc/firefly/organizations/peerOrganizations/org1.example.com
ORDERER_DIR=/etc/firefly/organizations/ordererOrganizations/example.com

ENV_VARS="\
  -e CORE_PEER_ADDRESS=fabric_peer:7051 \
  -e CORE_PEER_TLS_ENABLED=true \
  -e CORE_PEER_TLS_ROOTCERT_FILE=${ORG_DIR}/peers/fabric_peer.org1.example.com/tls/ca.crt \
  -e CORE_PEER_LOCALMSPID=${ORG_NAME} \
  -e CORE_PEER_MSPCONFIGPATH=${ORG_DIR}/users/Admin@org1.example.com/msp \
"

VOLUMES="\
  -v ${CHAINCODE}:/chaincode-go \
  -v ${STACK_NAME}_firefly_fabric:/etc/firefly \
"

CA_PARAMS="--tls --cafile ${ORDERER_DIR}/orderers/fabric_orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
RUN="docker run --rm --network=${NETWORK} ${ENV_VARS} ${VOLUMES} hyperledger/fabric-tools:2.4"

echo "Using name ${CHAINCODE_NAME}"
echo "Installing chaincode from ${CHAINCODE}..."
${RUN} /bin/bash -c "\
  peer lifecycle chaincode package /root/pkg.tar.gz --path /chaincode-go --lang golang --label ${CHAINCODE_NAME} && \
  peer lifecycle chaincode install /root/pkg.tar.gz \
"

PKG_ID=$(${RUN} peer lifecycle chaincode queryinstalled | grep -o "${CHAINCODE_NAME}:[^,]*")
if [ -z "$PKG_ID" ]; then
  echo "Error installing chaincode"
  exit 1
fi

echo "Package ID: ${PKG_ID}"

COMMITTED=$(\
  ${RUN} peer lifecycle chaincode querycommitted \
  --channelID ${CHANNEL} --name ${CHAINCODE_NAME} ${CA_PARAMS} 2>/dev/null | \
  grep -o "Version: ${CHAINCODE_VERSION}, ")

if [ -z "$COMMITTED" ]; then
  APPROVED=$(\
    ${RUN} peer lifecycle chaincode checkcommitreadiness \
    --channelID ${CHANNEL} --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --sequence 1 ${CA_PARAMS} 2>/dev/null | \
    grep -o "${ORG_NAME}: true")

  if [ -z "$APPROVED" ]; then
    echo "Approving chaincode..."
    ${RUN} peer lifecycle chaincode approveformyorg \
      --channelID ${CHANNEL} --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --sequence 1 --package-id ${PKG_ID} \
      -o fabric_orderer:7050 --ordererTLSHostnameOverride fabric_orderer ${CA_PARAMS}
  fi

  echo "Committing chaincode..."
  ${RUN} peer lifecycle chaincode commit \
    --channelID ${CHANNEL} --name ${CHAINCODE_NAME} --version ${CHAINCODE_VERSION} --sequence 1 \
    -o fabric_orderer:7050 --ordererTLSHostnameOverride fabric_orderer ${CA_PARAMS}

  if [ $? -ne 0 ]; then
    echo "Failed to commit"
    exit 1
  fi
else
  echo "Already committed"
fi

echo "Complete"
