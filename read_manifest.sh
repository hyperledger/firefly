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

 if [[ ! -x `which jq` ]]; then echo "Please install \"jq\" to continue"; exit 1; fi

FIREFLY_BUILDER_TAG=$(cat manifest.json | jq -r '.build."firefly-builder".image')
FABRIC_BUILDER_TAG=$(cat manifest.json | jq -r '.build."fabric-builder".image')
FABRIC_BUILDER_PLATFORM=$(cat manifest.json | jq -r '.build."fabric-builder".platform')
SOLIDITY_BUILDER_TAG=$(cat manifest.json | jq -r '.build."solidity-builder".image')
BASE_TAG=$(cat manifest.json | jq -r '.build.base.image')
UI_TAG=$(cat manifest.json | jq -r '.ui.tag')
UI_RELEASE=$(cat manifest.json | jq -r '.ui.release')

export FIREFLY_BUILDER_TAG=$FIREFLY_BUILDER_TAG
export FABRIC_BUILDER_TAG=$FABRIC_BUILDER_TAG
export FABRIC_BUILDER_PLATFORM=$FABRIC_BUILDER_PLATFORM
export SOLIDITY_BUILDER_TAG=$SOLIDITY_BUILDER_TAG
export BASE_TAG=$BASE_TAG
export UI_TAG=$UI_TAG
export UI_RELEASE=$UI_RELEASE