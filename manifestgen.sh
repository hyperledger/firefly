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

# This script will automatically update the manifest.yaml file with the
# latest releases of all FireFly microservice dependencies

rm -f manifest.yaml

echo $'# SPDX-License-Identifier: Apache-2.0\n' >> manifest.yaml
echo $'# FireFly Version Manifest' >> manifest.yaml
echo $'# This file describes the version of each microservice that should be used with this version of FireFly\n' >> manifest.yaml

declare -a arr=(
    "ethconnect"
    "fabconnect"
    "data-exchange-https"
    "tokens-erc-1155"
)

for i in "${arr[@]}"
do
   echo "$i:" >> manifest.yaml
   echo "  image: ghcr.io/hyperledger/firefly-$i" >> manifest.yaml
   echo "  tag: "$(curl https://api.github.com/repos/hyperledger/firefly-$i/releases/latest -s | jq .tag_name -r)"" >> manifest.yaml
done