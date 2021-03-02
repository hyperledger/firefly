package io.kaleido.kat.states;

import io.kaleido.kat.contracts.AssetTrailContract;
import net.corda.core.contracts.BelongsToContract;
import net.corda.core.identity.AbstractParty;
import net.corda.core.identity.Party;
import org.jetbrains.annotations.NotNull;

import java.util.ArrayList;
import java.util.List;
import java.util.stream.Collectors;

@BelongsToContract(AssetTrailContract.class)
public class AssetInstanceBatchCreated implements AssetEventState {
    private final Party author;
    private final String batchHash;
    private final List<Party> assetParticipants;

    public AssetInstanceBatchCreated(Party author, String batchHash, List<Party> assetParticipants) {
        this.author = author;
        this.batchHash = batchHash;
        this.assetParticipants = assetParticipants;
    }

    @NotNull
    @Override
    public List<AbstractParty> getParticipants() {
        return new ArrayList<>(assetParticipants);
    }

    @Override
    public String toString() {
        return String.format("AssetInstanceBatchCreated(author=%s, batchHash=%s, participants=%s)", author, batchHash, assetParticipants);
    }

    @Override
    public Party getAuthor() {
        return author;
    }

    @Override
    public List<Party> getAssetParticipants() {
        return assetParticipants;
    }

    public String getBatchHash() {
        return batchHash;
    }
}
