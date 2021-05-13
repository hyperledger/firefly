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

import io.kaleido.kat.states.DescribedAssetInstanceCreated;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.InitiatingFlow;
import net.corda.core.flows.StartableByRPC;
import net.corda.core.identity.Party;

import java.util.ArrayList;
import java.util.List;


@StartableByRPC
@InitiatingFlow
public class CreateDescribedAssetInstanceFlow extends CreateAssetEventFlow<DescribedAssetInstanceCreated>{
    private final String assetInstanceID;
    private final String assetDefinitionID;
    private final String descriptionHash;
    private final String contentHash;

    public CreateDescribedAssetInstanceFlow(String assetInstanceID, String assetDefinitionID, String descriptionHash, String contentHash, List<Party> observers) {
        super(observers);
        this.assetInstanceID = assetInstanceID;
        this.assetDefinitionID = assetDefinitionID;
        this.descriptionHash = descriptionHash;
        this.contentHash = contentHash;
    }

    @Override
    public DescribedAssetInstanceCreated getAssetEvent() {
        List<Party> participants = new ArrayList<>(this.observers);
        participants.add(getOurIdentity());
        return new DescribedAssetInstanceCreated(assetInstanceID, assetDefinitionID,getOurIdentity(), descriptionHash, contentHash,participants);
    }
}
