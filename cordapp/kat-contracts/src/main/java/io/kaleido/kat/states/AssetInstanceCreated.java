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
