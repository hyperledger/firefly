package io.kaleido.kat.flows;

import io.kaleido.kat.states.DescribedStructuredAssetDefinitionCreated;
import net.corda.core.flows.FlowSession;
import net.corda.core.flows.InitiatedBy;

@InitiatedBy(CreateDescribedStructuredAssetDefinitionFlow.class)
public class ReceiveDescribedStructuredAssetDefinitionFlow extends ReceiveAssetEventFlow<DescribedStructuredAssetDefinitionCreated>{
    public ReceiveDescribedStructuredAssetDefinitionFlow(FlowSession otherPartySession) {
        super(otherPartySession);
    }
}
