package io.kaleido.kat.flows;
import io.kaleido.kat.states.DescribedStructuredAssetDefinitionCreated;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.InitiatingFlow;
import net.corda.core.flows.StartableByRPC;
import net.corda.core.identity.Party;

import java.util.List;

@InitiatingFlow
@StartableByRPC
public class CreateDescribedStructuredAssetDefinitionFlow extends CreateAssetEventFlow<DescribedStructuredAssetDefinitionCreated>{
    private final String assetDefinitionID;
    private final String name;
    private final boolean isContentPrivate;
    private final boolean isContentUnique;
    private final String descriptionSchemaHash;
    private final String contentSchemaHash;

    public CreateDescribedStructuredAssetDefinitionFlow(String assetDefinitionID, String name, boolean isContentPrivate, boolean isContentUnique, String descriptionSchemaHash, String contentSchemaHash, List<Party> observers, UniqueIdentifier orderingContext) {
        super(observers, orderingContext);
        this.assetDefinitionID = assetDefinitionID;
        this.name = name;
        this.isContentPrivate = isContentPrivate;
        this.isContentUnique = isContentUnique;
        this.descriptionSchemaHash = descriptionSchemaHash;
        this.contentSchemaHash = contentSchemaHash;
    }

    @Override
    public DescribedStructuredAssetDefinitionCreated getAssetEvent(){
        return new DescribedStructuredAssetDefinitionCreated(assetDefinitionID, getOurIdentity(),name, isContentPrivate, isContentUnique, descriptionSchemaHash, contentSchemaHash);
    }
}
