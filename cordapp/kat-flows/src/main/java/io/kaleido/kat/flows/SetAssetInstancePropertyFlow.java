package io.kaleido.kat.flows;

import io.kaleido.kat.states.AssetInstancePropertySet;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.InitiatingFlow;
import net.corda.core.flows.StartableByRPC;
import net.corda.core.identity.Party;

import java.util.ArrayList;
import java.util.List;

@StartableByRPC
@InitiatingFlow
public class SetAssetInstancePropertyFlow extends CreateAssetEventFlow<AssetInstancePropertySet>{
    private final String assetDefinitionID;
    private final String assetInstanceID;
    private final String key;
    private final String value;

    public SetAssetInstancePropertyFlow(String assetDefinitionID, String assetInstanceID, String key, String value, List<Party> observers) {
        super(observers);
        this.assetDefinitionID = assetDefinitionID;
        this.assetInstanceID = assetInstanceID;
        this.key = key;
        this.value = value;
    }

    @Override
    public AssetInstancePropertySet getAssetEvent(){
        List<Party> participants = new ArrayList<>(this.observers);
        participants.add(getOurIdentity());
        return new AssetInstancePropertySet(assetDefinitionID, assetInstanceID, getOurIdentity(), key, value, participants);
    }
}
