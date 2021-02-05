package io.kaleido.kat.states;

import io.kaleido.kat.contracts.KatContract;
import net.corda.core.contracts.BelongsToContract;
import net.corda.core.contracts.ContractState;
import net.corda.core.identity.AbstractParty;
import net.corda.core.identity.Party;
import org.jetbrains.annotations.NotNull;

import java.util.List;

@BelongsToContract(KatContract.class)
public class DescribedUnstructuredAssetDefinitionCreated implements AssetEventState {
    private final String assetDefinitionID;
    private final Party author;
    private final String name;
    private final boolean isContentPrivate;
    private final boolean isContentUnique;
    private final String descriptionSchemaHash;

    public DescribedUnstructuredAssetDefinitionCreated(String assetDefinitionID, Party author, String name, boolean isContentPrivate, boolean isContentUnique, String descriptionSchemaHash) {
        this.assetDefinitionID = assetDefinitionID;
        this.author = author;
        this.name = name;
        this.isContentPrivate = isContentPrivate;
        this.isContentUnique = isContentUnique;
        this.descriptionSchemaHash = descriptionSchemaHash;
    }

    @NotNull
    @Override
    public List<AbstractParty> getParticipants() {
        return List.of(author);
    }

    @Override
    public String toString() {
        return String.format("DescribedUnstructuredAssetDefinitionCreated(assetDefinitionID=%s, author=%s, name=%s, isContentPrivate=%s, isContentUnique=%s, descriptionSchemaHash=%s)", assetDefinitionID, author, name, isContentPrivate, isContentUnique, descriptionSchemaHash);
    }

    @Override
    public Party getAuthor() {
        return author;
    }
}
