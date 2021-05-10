package io.kaleido.kat.flows;

import io.kaleido.kat.states.DescribedAssetInstanceCreated;
import net.corda.core.flows.FlowSession;
import net.corda.core.flows.InitiatedBy;

@InitiatedBy(CreateDescribedAssetInstanceFlow.class)
public class ReceiveDescribedAssetInstanceFlow extends ReceiveAssetEventFlow<DescribedAssetInstanceCreated>{
    public ReceiveDescribedAssetInstanceFlow(FlowSession otherPartySession) {
        super(otherPartySession);
    }

}
