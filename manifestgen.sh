#!/bin/bash

# Copyright © 2021 Kaleido, Inc.
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

# This script will automatically update the manifest.json file with the
# latest releases of all FireFly microservice dependencies

rm -f manifest.json

SERVICES=(
    "ethconnect"
    "fabconnect"
    "data-exchange-https"
    "tokens-erc-1155"
)
SERVICE_COUNT=${#SERVICES[@]}

echo "{" >> manifest.json

for (( i=0; i<${SERVICE_COUNT}; i++ ))
do
    echo "  \"${SERVICES[$i]}\": {" >> manifest.json
    echo "    \"image\": \"ghcr.io/hyperledger/firefly-${SERVICES[$i]}\"," >> manifest.json
    echo "    \"tag\": \""$(curl https://api.github.com/repos/hyperledger/firefly-${SERVICES[$i]}/releases/latest -s | jq .tag_name -r)\""" >> manifest.json
    if [[ $(($i + 1)) -eq ${SERVICE_COUNT} ]]
    then
        echo "  }" >> manifest.json
    else
        echo "  }," >> manifest.json
    fi
done

echo "}" >> manifest.json