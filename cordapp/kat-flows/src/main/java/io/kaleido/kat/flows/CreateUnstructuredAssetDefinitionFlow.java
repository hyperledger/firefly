package io.kaleido.kat.flows;

import io.kaleido.kat.states.UnstructuredAssetDefinitionCreated;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.InitiatingFlow;
import net.corda.core.flows.StartableByRPC;
import net.corda.core.identity.Party;

import java.util.List;

@InitiatingFlow
@StartableByRPC
public class CreateUnstructuredAssetDefinitionFlow extends CreateAssetEventFlow<UnstructuredAssetDefinitionCreated>{
    private final String assetDefinitionID;
    private final String name;
    private final boolean isContentPrivate;
    private final boolean isContentUnique;

    public CreateUnstructuredAssetDefinitionFlow(String assetDefinitionID, String name, boolean isContentPrivate, boolean isContentUnique, List<Party> observers, UniqueIdentifier orderingContext) {
        super(observers, orderingContext);
        this.assetDefinitionID = assetDefinitionID;
        this.name = name;
        this.isContentPrivate = isContentPrivate;
        this.isContentUnique = isContentUnique;
    }

    @Override
    public UnstructuredAssetDefinitionCreated getAssetEvent() {
        return new UnstructuredAssetDefinitionCreated(assetDefinitionID, getOurIdentity(), name, isContentPrivate, isContentUnique);
    }
}
