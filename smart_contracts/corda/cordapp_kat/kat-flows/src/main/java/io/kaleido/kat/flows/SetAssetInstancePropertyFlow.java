// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package io.kaleido.kat.flows;

import io.kaleido.kat.states.AssetInstancePropertySet;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.InitiatingFlow;
import net.corda.core.flows.StartableByRPC;
import net.corda.core.identity.Party;

import java.util.ArrayList;
import java.util.List;

@StartableByRPC
@InitiatingFlow
public class SetAssetInstancePropertyFlow extends CreateAssetEventFlow<AssetInstancePropertySet>{
    private final String assetDefinitionID;
    private final String assetInstanceID;
    private final String key;
    private final String value;

    public SetAssetInstancePropertyFlow(String assetDefinitionID, String assetInstanceID, String key, String value, List<Party> observers) {
        super(observers);
        this.assetDefinitionID = assetDefinitionID;
        this.assetInstanceID = assetInstanceID;
        this.key = key;
        this.value = value;
    }

    @Override
    public AssetInstancePropertySet getAssetEvent(){
        List<Party> participants = new ArrayList<>(this.observers);
        participants.add(getOurIdentity());
        return new AssetInstancePropertySet(assetDefinitionID, assetInstanceID, getOurIdentity(), key, value, participants);
    }
}
