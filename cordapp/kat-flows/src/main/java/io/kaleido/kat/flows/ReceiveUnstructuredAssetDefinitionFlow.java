package io.kaleido.kat.flows;

import io.kaleido.kat.states.UnstructuredAssetDefinitionCreated;
import net.corda.core.flows.FlowSession;
import net.corda.core.flows.InitiatedBy;

@InitiatedBy(CreateUnstructuredAssetDefinitionFlow.class)
public class ReceiveUnstructuredAssetDefinitionFlow extends ReceiveAssetEventFlow<UnstructuredAssetDefinitionCreated> {
    public ReceiveUnstructuredAssetDefinitionFlow(FlowSession otherPartySession) {
        super(otherPartySession);
    }
}
