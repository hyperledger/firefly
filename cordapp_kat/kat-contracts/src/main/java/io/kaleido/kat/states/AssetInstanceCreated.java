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

package io.kaleido.kat.states;

import io.kaleido.kat.contracts.AssetTrailContract;
import net.corda.core.contracts.BelongsToContract;
import net.corda.core.identity.AbstractParty;
import net.corda.core.identity.Party;
import org.jetbrains.annotations.NotNull;

import java.util.ArrayList;
import java.util.List;

@BelongsToContract(AssetTrailContract.class)
public class AssetInstanceCreated implements AssetEventState {
    private final String assetInstanceID;
    private final String assetDefinitionID;
    private final Party author;
    private final String contentHash;
    private final List<Party> participants;

    public AssetInstanceCreated(String assetInstanceID, String assetDefinitionID, Party author, String contentHash, List<Party> participants) {
        this.assetInstanceID = assetInstanceID;
        this.assetDefinitionID = assetDefinitionID;
        this.author = author;
        this.contentHash = contentHash;
        this.participants = participants;
    }

    @NotNull
    @Override
    public List<AbstractParty> getParticipants() {
        return new ArrayList<>(participants);
    }

    @Override
    public String toString() {
        return String.format("AssetInstanceCreated(assetInstanceID=%s, assetDefinitionID=%s, author=%s, contentHash=%s, participants=%s)", assetInstanceID, assetDefinitionID, author, contentHash, participants);
    }

    @Override
    public Party getAuthor() {
        return author;
    }

    public String getAssetInstanceID() {
        return assetInstanceID;
    }

    public String getAssetDefinitionID() {
        return assetDefinitionID;
    }

    public String getContentHash() {
        return contentHash;
    }
}
