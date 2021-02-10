package io.kaleido.kat.flows;

import io.kaleido.kat.states.StructuredAssetDefinitionCreated;
import net.corda.core.flows.FlowSession;
import net.corda.core.flows.InitiatedBy;

@InitiatedBy(CreateStructuredAssetDefinitionFlow.class)
public class ReceiveStructuredAssetDefinitionFlow extends ReceiveAssetEventFlow<StructuredAssetDefinitionCreated> {
    public ReceiveStructuredAssetDefinitionFlow(FlowSession otherPartySession) {
        super(otherPartySession);
    }
}
