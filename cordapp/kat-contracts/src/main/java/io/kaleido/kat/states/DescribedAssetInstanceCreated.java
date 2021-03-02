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

    public DescribedAssetInstanceCreated(String assetInstanceID, String assetDefinitionID, Party author, String descriptionHash, String contentHash, List<Party> otherParties) {
        this.assetInstanceID = assetInstanceID;
        this.assetDefinitionID = assetDefinitionID;
        this.author = author;
        this.descriptionHash = descriptionHash;
        this.contentHash = contentHash;
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
        return String.format("DescribedAssetInstanceCreated(assetInstanceID=%s, assetDefinitionID=%s, author=%s, descriptionHash=%s, contentHash=%s)", assetInstanceID, assetDefinitionID, author, descriptionHash, contentHash);
    }

    @Override
    public Party getAuthor() {
        return author;
    }

    @Override
    public List<Party> getAssetParticipants() {
        return participants;
    }
}
