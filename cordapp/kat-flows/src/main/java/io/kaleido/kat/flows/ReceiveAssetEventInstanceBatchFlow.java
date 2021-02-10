package io.kaleido.kat.flows;

import io.kaleido.kat.states.AssetInstanceBatchCreated;
import net.corda.core.flows.FlowSession;
import net.corda.core.flows.InitiatedBy;

@InitiatedBy(CreateAssetInstanceBatchFlow.class)
public class ReceiveAssetEventInstanceBatchFlow extends ReceiveAssetEventFlow<AssetInstanceBatchCreated> {
    public ReceiveAssetEventInstanceBatchFlow(FlowSession otherPartySession) {
        super(otherPartySession);
    }
}
