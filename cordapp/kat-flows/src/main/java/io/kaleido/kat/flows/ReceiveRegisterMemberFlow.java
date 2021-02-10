package io.kaleido.kat.flows;

import io.kaleido.kat.states.MemberRegistered;
import net.corda.core.flows.FlowSession;
import net.corda.core.flows.InitiatedBy;

@InitiatedBy(RegisterMemberFlow.class)
public class ReceiveRegisterMemberFlow extends ReceiveAssetEventFlow<MemberRegistered> {
    public ReceiveRegisterMemberFlow(FlowSession otherPartySession) {
        super(otherPartySession);
    }
}
