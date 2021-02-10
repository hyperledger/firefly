package io.kaleido.kat.states;

import io.kaleido.kat.contracts.KatContract;
import net.corda.core.contracts.BelongsToContract;
import net.corda.core.contracts.ContractState;
import net.corda.core.identity.AbstractParty;
import net.corda.core.identity.Party;
import org.jetbrains.annotations.NotNull;

import java.util.List;

@BelongsToContract(KatContract.class)
public class AssetInstanceBatchCreated implements   AssetEventState {
    private final Party author;
    private final String batchHash;

    public AssetInstanceBatchCreated(Party author, String batchHash) {
        this.author = author;
        this.batchHash = batchHash;
    }

    @NotNull
    @Override
    public List<AbstractParty> getParticipants() {
        return List.of(author);
    }

    @Override
    public String toString() {
        return String.format("AssetInstanceBatchCreated(author=%s, batchHash=%s)", author, batchHash);
    }

    @Override
    public Party getAuthor() {
        return author;
    }

    public String getBatchHash() {
        return batchHash;
    }
}
