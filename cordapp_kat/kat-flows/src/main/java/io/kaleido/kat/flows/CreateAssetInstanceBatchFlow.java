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

import io.kaleido.kat.states.AssetInstanceBatchCreated;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.*;
import net.corda.core.identity.Party;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.ArrayList;
import java.util.List;

@InitiatingFlow
@StartableByRPC
public class CreateAssetInstanceBatchFlow extends CreateAssetEventFlow<AssetInstanceBatchCreated> {
    private final String batchHash;
    public CreateAssetInstanceBatchFlow(String batchHash, List<Party> observers) {
        super(observers);
        this.batchHash = batchHash;
    }

    @Override
    public AssetInstanceBatchCreated getAssetEvent(){
        List<Party> participants = new ArrayList<>(this.observers);
        participants.add(getOurIdentity());
        return new AssetInstanceBatchCreated(getOurIdentity(), batchHash, participants);
    }
}
