package io.kaleido.kat.flows;

import co.paralleluniverse.fibers.Suspendable;
import net.corda.core.flows.*;
import net.corda.core.node.StatesToRecord;

@InitiatedBy(SendTxToObserverNodeFlow.class)
public class ReceiveTxAsObserverNodeFlow extends FlowLogic<Void> {
    private final FlowSession otherSideSession;

    public ReceiveTxAsObserverNodeFlow(FlowSession otherSideSession) {
        this.otherSideSession = otherSideSession;
    }

    @Suspendable
    @Override
    public Void call() throws FlowException {
        subFlow(new ReceiveTransactionFlow(otherSideSession, true, StatesToRecord.ALL_VISIBLE));
        return null;
    }
}
