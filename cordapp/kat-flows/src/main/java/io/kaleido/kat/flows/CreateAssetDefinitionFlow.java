package io.kaleido.kat.flows;
import io.kaleido.kat.states.AssetDefinitionCreated;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.InitiatingFlow;
import net.corda.core.flows.StartableByRPC;
import net.corda.core.identity.Party;

import java.util.List;

@InitiatingFlow
@StartableByRPC
public class CreateAssetDefinitionFlow extends CreateAssetEventFlow<AssetDefinitionCreated>{
    private final String assetDefinitionHash;

    public CreateAssetDefinitionFlow(String assetDefinitionHash,  List<Party> observers, UniqueIdentifier orderingContext) {
        super(observers, orderingContext);
        this.assetDefinitionHash = assetDefinitionHash;
    }

    @Override
    public AssetDefinitionCreated getAssetEvent(){
        return new AssetDefinitionCreated(assetDefinitionHash, getOurIdentity());
    }
}
