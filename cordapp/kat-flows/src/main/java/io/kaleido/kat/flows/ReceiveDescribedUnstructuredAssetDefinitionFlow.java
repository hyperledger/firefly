package io.kaleido.kat.flows;

import io.kaleido.kat.states.DescribedUnstructuredAssetDefinitionCreated;
import net.corda.core.flows.FlowSession;
import net.corda.core.flows.InitiatedBy;

@InitiatedBy(CreateDescribedUnstructuredAssetDefinitionFlow.class)
public class ReceiveDescribedUnstructuredAssetDefinitionFlow extends ReceiveAssetEventFlow<DescribedUnstructuredAssetDefinitionCreated>{
    public ReceiveDescribedUnstructuredAssetDefinitionFlow(FlowSession otherPartySession) {
        super(otherPartySession);
    }
}
