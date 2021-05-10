package io.kaleido.kat.flows;

import io.kaleido.kat.states.AssetInstanceCreated;
import net.corda.core.flows.FlowSession;
import net.corda.core.flows.InitiatedBy;

@InitiatedBy(CreateAssetInstanceFlow.class)
public class ReceiveAssetInstanceFlow extends ReceiveAssetEventFlow<AssetInstanceCreated>{
    public ReceiveAssetInstanceFlow(FlowSession otherPartySession) {
        super(otherPartySession);
    }
}
