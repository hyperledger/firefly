package io.kaleido.kat.flows;

import io.kaleido.kat.states.AssetInstancePropertySet;
import net.corda.core.flows.FlowSession;
import net.corda.core.flows.InitiatedBy;

@InitiatedBy(SetAssetInstancePropertyFlow.class)
public class ReceiveSetAssetInstancePropertyFlow extends ReceiveAssetEventFlow<AssetInstancePropertySet> {
    public ReceiveSetAssetInstancePropertyFlow(FlowSession otherPartySession) {
        super(otherPartySession);
    }
}
