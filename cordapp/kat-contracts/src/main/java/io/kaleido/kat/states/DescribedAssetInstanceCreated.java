package io.kaleido.kat.states;

import io.kaleido.kat.contracts.KatContract;
import net.corda.core.contracts.BelongsToContract;
import net.corda.core.contracts.ContractState;
import net.corda.core.identity.AbstractParty;
import net.corda.core.identity.Party;
import org.jetbrains.annotations.NotNull;

import java.util.List;

@BelongsToContract(KatContract.class)
public class DescribedAssetInstanceCreated implements AssetEventState {
    private final String assetInstanceID;
    private final String assetDefinitionID;
    private final Party author;
    private final String descriptionHash;
    private final String contentHash;

    public DescribedAssetInstanceCreated(String assetInstanceID, String assetDefinitionID, Party author, String descriptionHash, String contentHash) {
        this.assetInstanceID = assetInstanceID;
        this.assetDefinitionID = assetDefinitionID;
        this.author = author;
        this.descriptionHash = descriptionHash;
        this.contentHash = contentHash;
    }

    @NotNull
    @Override
    public List<AbstractParty> getParticipants() {
        return List.of(author);
    }

    @Override
    public String toString() {
        return String.format("DescribedAssetInstanceCreated(assetInstanceID=%s, assetDefinitionID=%s, author=%s, descriptionHash=%s, contentHash=%s)", assetInstanceID, assetDefinitionID, author, descriptionHash, contentHash);
    }

    @Override
    public Party getAuthor() {
        return author;
    }
}
