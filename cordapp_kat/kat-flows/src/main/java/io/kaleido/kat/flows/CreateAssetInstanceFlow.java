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

import io.kaleido.kat.states.AssetInstanceCreated;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.InitiatingFlow;
import net.corda.core.flows.StartableByRPC;
import net.corda.core.identity.Party;

import java.util.ArrayList;
import java.util.List;

@InitiatingFlow
@StartableByRPC
public class CreateAssetInstanceFlow extends CreateAssetEventFlow<AssetInstanceCreated>{
    private final String assetInstanceID;
    private final String assetDefinitionID;
    private final String contentHash;

    public CreateAssetInstanceFlow(String assetInstanceID, String assetDefinitionID, String contentHash, List<Party> observers) {
        super(observers);
        this.assetInstanceID = assetInstanceID;
        this.assetDefinitionID = assetDefinitionID;
        this.contentHash = contentHash;
    }

    @Override
    public AssetInstanceCreated getAssetEvent(){
        List<Party> participants = new ArrayList<>(this.observers);
        participants.add(getOurIdentity());
        return new AssetInstanceCreated(assetInstanceID, assetDefinitionID, getOurIdentity(), contentHash, participants);
    }
}
