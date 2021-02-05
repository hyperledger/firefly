package io.kaleido.kat.flows;

import io.kaleido.kat.states.StructuredAssetDefinitionCreated;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.InitiatingFlow;
import net.corda.core.flows.StartableByRPC;
import net.corda.core.identity.Party;

import java.util.List;

@InitiatingFlow
@StartableByRPC
public class CreateStructuredAssetDefinitionFlow extends CreateAssetEventFlow<StructuredAssetDefinitionCreated>{
    private final String assetDefinitionID;
    private final String name;
    private final boolean isContentPrivate;
    private final boolean isContentUnique;
    private final String contentSchemaHash;

    public CreateStructuredAssetDefinitionFlow(String assetDefinitionID, String name, boolean isContentPrivate, boolean isContentUnique, String contentSchemaHash, List<Party> observers, UniqueIdentifier orderingContext) {
        super(observers, orderingContext);
        this.assetDefinitionID = assetDefinitionID;
        this.name = name;
        this.isContentPrivate = isContentPrivate;
        this.isContentUnique = isContentUnique;
        this.contentSchemaHash = contentSchemaHash;
    }

    @Override
    public StructuredAssetDefinitionCreated getAssetEvent() {
        return new StructuredAssetDefinitionCreated(assetDefinitionID, getOurIdentity(), name, isContentPrivate, isContentUnique, contentSchemaHash);
    }
}
