package io.kaleido.kat.flows;


import io.kaleido.kat.states.AssetDefinitionCreated;
import net.corda.core.flows.FlowSession;
import net.corda.core.flows.InitiatedBy;

@InitiatedBy(ShareExistingAssetDefinitionFlow.class)
public class ReceiveExistingAssetDefinitionFlow extends ReceiveAssetEventFlow<AssetDefinitionCreated>{
    public ReceiveExistingAssetDefinitionFlow(FlowSession otherPartySession) {
        super(otherPartySession);
    }
}
