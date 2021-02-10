package io.kaleido.kat.flows;

import co.paralleluniverse.fibers.Suspendable;
import net.corda.core.contracts.ContractState;
import net.corda.core.flows.*;
import net.corda.core.node.StatesToRecord;


public class ReceiveAssetEventFlow<T extends ContractState> extends FlowLogic<Void> {
    private final FlowSession otherPartySession;

    public ReceiveAssetEventFlow(FlowSession otherPartySession) {
        this.otherPartySession = otherPartySession;
    }

    @Suspendable
    @Override
    public Void call() throws FlowException {
        subFlow(new ReceiveTransactionFlow(otherPartySession, true, StatesToRecord.ALL_VISIBLE));
        return null;
    }
}
