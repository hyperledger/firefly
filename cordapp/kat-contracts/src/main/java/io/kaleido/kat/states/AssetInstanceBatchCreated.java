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
    private final List<Party> participants;

    public AssetInstanceBatchCreated(Party author, String batchHash, List<Party> otherParties) {
        this.author = author;
        this.batchHash = batchHash;
        this.participants = otherParties;
        this.participants.add(author);
    }

    @NotNull
    @Override
    public List<AbstractParty> getParticipants() {
        return new ArrayList<>(participants);
    }

    @Override
    public String toString() {
        return String.format("AssetInstanceBatchCreated(author=%s, batchHash=%s)", author, batchHash);
    }

    @Override
    public Party getAuthor() {
        return author;
    }

    @Override
    public List<Party> getAssetParticipants() {
        return participants;
    }

    public String getBatchHash() {
        return batchHash;
    }
}
