package io.kaleido.kat.flows;
import io.kaleido.kat.states.AssetDefinitionCreated;
import net.corda.core.flows.FlowSession;
import net.corda.core.flows.InitiatedBy;

@InitiatedBy(CreateAssetDefinitionFlow.class)
public class ReceiveAssetDefinitionFlow extends ReceiveAssetEventFlow<AssetDefinitionCreated>{
    public ReceiveAssetDefinitionFlow(FlowSession otherPartySession) {
        super(otherPartySession);
    }
}
