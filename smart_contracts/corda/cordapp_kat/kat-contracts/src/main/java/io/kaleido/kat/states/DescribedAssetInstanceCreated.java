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
public class DescribedAssetInstanceCreated implements AssetEventState {
    private final String assetInstanceID;
    private final String assetDefinitionID;
    private final Party author;
    private final String descriptionHash;
    private final String contentHash;
    private final List<Party> participants;

    public DescribedAssetInstanceCreated(String assetInstanceID, String assetDefinitionID, Party author, String descriptionHash, String contentHash, List<Party> participants) {
        this.assetInstanceID = assetInstanceID;
        this.assetDefinitionID = assetDefinitionID;
        this.author = author;
        this.descriptionHash = descriptionHash;
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
        return String.format("DescribedAssetInstanceCreated(assetInstanceID=%s, assetDefinitionID=%s, author=%s, descriptionHash=%s, contentHash=%s, participants=%s)", assetInstanceID, assetDefinitionID, author, descriptionHash, contentHash, participants);
    }

    @Override
    public Party getAuthor() {
        return author;
    }
}
