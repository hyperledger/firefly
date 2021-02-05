package io.kaleido.kat.flows;

import io.kaleido.kat.states.DescribedAssetInstanceCreated;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.InitiatingFlow;
import net.corda.core.flows.StartableByRPC;
import net.corda.core.identity.Party;

import java.util.List;

@InitiatingFlow
@StartableByRPC
public class CreateDescribedAssetInstanceFlow extends CreateAssetEventFlow<DescribedAssetInstanceCreated>{
    private final String assetInstanceID;
    private final String assetDefinitionID;
    private final String descriptionHash;
    private final String contentHash;

    public CreateDescribedAssetInstanceFlow(String assetInstanceID, String assetDefinitionID, String descriptionHash, String contentHash, List<Party> observers, UniqueIdentifier orderingContext) {
        super(observers, orderingContext);
        this.assetInstanceID = assetInstanceID;
        this.assetDefinitionID = assetDefinitionID;
        this.descriptionHash = descriptionHash;
        this.contentHash = contentHash;
    }

    @Override
    public DescribedAssetInstanceCreated getAssetEvent() {
        return new DescribedAssetInstanceCreated(assetInstanceID, assetDefinitionID, getOurIdentity(), descriptionHash, contentHash);
    }
}
