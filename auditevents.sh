#!/bin/bash

# Copyright © 2022 Kaleido, Inc.
#
# SPDX-License-Identifier: Apache-2.0
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script will query all events of type blockchain_event_received, and
# validate that the blockchain event "protocolId" is continuously increasing.

set -euo pipefail

if [[ ! -x `which jq` ]]; then echo "Please install \"jq\" to continue"; exit 1; fi

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <host>"
  exit 1
fi
HOST=$1

blockchainType=$(curl -s $HOST/api/v1/status | jq ".plugins.blockchain[0].pluginType")

expectedSource="${blockchainType}"
expectedListener="null"

limit=50
received=$limit
validated=0

lastSequence=0
lastProtocolId=

echo "Checking all blockchain events for increasing protocol ID."
echo
echo "Configuration:"
echo "  host=${HOST}"
echo "  source=${expectedSource}"
echo "  listener=${expectedListener}"
echo
printf "%-10s %s\n" "Sequence" "Protocol ID"
while [[ "$received" -eq "$limit" ]]; do
  seqQuery=">${lastSequence}"
  events=$(curl -s "$HOST/api/v1/events?type=blockchain_event_received&fetchreferences&sort=sequence&sequence=${seqQuery}&limit=${limit}")
  received=$(jq ". | length" <<< "$events")
  for i in $(seq "$received"); do
    event=$(jq ".[$((i - 1))]" <<< "$events")
    sequence=$(jq ".sequence" <<< "$event")
    protocolId=$(jq -r ".blockchainEvent.protocolId" <<< "$event")
    listener=$(jq ".blockchainEvent.listener" <<< "$event")
    src=$(jq ".blockchainEvent.source" <<< "$event")
    if [[ "$listener" != "$expectedListener" || "$src" != "$expectedSource" ]]; then
      continue
    fi
    printf "%-10s %s\n" "$sequence" "$protocolId"
    if ! [[ "$protocolId" > "$lastProtocolId" ]]; then
      echo "Out of order events detected!"
      exit 1
    fi
    lastSequence=$sequence
    lastProtocolId=$protocolId
    validated=$((validated + 1))
  done
done
echo "${validated} events validated."
exit 0
