package io.kaleido.kat.states;

import io.kaleido.kat.contracts.KatContract;
import net.corda.core.contracts.BelongsToContract;
import net.corda.core.contracts.ContractState;
import net.corda.core.identity.AbstractParty;
import net.corda.core.identity.Party;
import org.jetbrains.annotations.NotNull;

import java.util.List;

@BelongsToContract(KatContract.class)
public class AssetInstancePropertySet implements AssetEventState {
    private final String assetInstanceID;
    private final Party author;
    private final String key;
    private final String value;

    public AssetInstancePropertySet(String assetInstanceID, Party author, String key, String value) {
        this.assetInstanceID = assetInstanceID;
        this.author = author;
        this.key = key;
        this.value = value;
    }

    @NotNull
    @Override
    public List<AbstractParty> getParticipants() {
        return List.of(author);
    }

    @Override
    public String toString() {
        return String.format("AssetInstancePropertySet(assetInstanceID=%s, author=%s, key=%s, value=%s)", assetInstanceID, author, key, value);
    }

    @Override
    public Party getAuthor() {
        return author;
    }

    public String getAssetInstanceID() {
        return assetInstanceID;
    }

    public String getKey() {
        return key;
    }

    public String getValue() {
        return value;
    }
}
