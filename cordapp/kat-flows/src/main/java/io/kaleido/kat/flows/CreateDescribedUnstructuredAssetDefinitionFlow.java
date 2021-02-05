package io.kaleido.kat.flows;

import io.kaleido.kat.states.DescribedUnstructuredAssetDefinitionCreated;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.InitiatingFlow;
import net.corda.core.flows.StartableByRPC;
import net.corda.core.identity.Party;

import java.util.List;

@InitiatingFlow
@StartableByRPC
public class CreateDescribedUnstructuredAssetDefinitionFlow extends CreateAssetEventFlow<DescribedUnstructuredAssetDefinitionCreated>{
    private final String assetDefinitionID;
    private final String name;
    private final boolean isContentPrivate;
    private final boolean isContentUnique;
    private final String descriptionSchemaHash;

    public CreateDescribedUnstructuredAssetDefinitionFlow(String assetDefinitionID, String name, boolean isContentPrivate, boolean isContentUnique, String descriptionSchemaHash, List<Party> observers, UniqueIdentifier orderingContext) {
        super(observers, orderingContext);
        this.assetDefinitionID = assetDefinitionID;
        this.name = name;
        this.isContentPrivate = isContentPrivate;
        this.isContentUnique = isContentUnique;
        this.descriptionSchemaHash = descriptionSchemaHash;
    }

    @Override
    public DescribedUnstructuredAssetDefinitionCreated getAssetEvent() {
        return new DescribedUnstructuredAssetDefinitionCreated(assetDefinitionID, getOurIdentity(), name, isContentPrivate, isContentUnique, descriptionSchemaHash);
    }
}
