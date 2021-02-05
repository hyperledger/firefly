package io.kaleido.kat.flows;

import co.paralleluniverse.fibers.Suspendable;
import net.corda.core.flows.*;
import net.corda.core.identity.Party;
import net.corda.core.transactions.SignedTransaction;

@InitiatingFlow
public class SendTxToObserverNodeFlow extends FlowLogic<Void> {
    private final Party observer;
    private final SignedTransaction finalTx;

    public SendTxToObserverNodeFlow(Party observer, SignedTransaction finalTx) {
        this.observer = observer;
        this.finalTx = finalTx;
    }

    @Suspendable
    @Override
    public Void call() throws FlowException {
        FlowSession session = initiateFlow(observer);
        return subFlow(new SendTransactionFlow(session, finalTx));
    }
}
