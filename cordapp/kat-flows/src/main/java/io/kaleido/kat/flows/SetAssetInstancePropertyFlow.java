package io.kaleido.kat.flows;

import io.kaleido.kat.states.AssetInstancePropertySet;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.InitiatingFlow;
import net.corda.core.flows.StartableByRPC;
import net.corda.core.identity.Party;

import java.util.List;

@StartableByRPC
@InitiatingFlow
public class SetAssetInstancePropertyFlow extends CreateAssetEventFlow<AssetInstancePropertySet>{
    private final String assetInstanceID;
    private final String key;
    private final String value;

    public SetAssetInstancePropertyFlow(String assetInstanceID, String key, String value, List<Party> observers, UniqueIdentifier orderingContext) {
        super(observers, orderingContext);
        this.assetInstanceID = assetInstanceID;
        this.key = key;
        this.value = value;
    }

    @Override
    public AssetInstancePropertySet getAssetEvent(){
        return new AssetInstancePropertySet(assetInstanceID, getOurIdentity(), key, value);
    }
}
