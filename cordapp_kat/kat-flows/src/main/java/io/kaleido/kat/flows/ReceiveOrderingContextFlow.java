package io.kaleido.kat.flows;


import co.paralleluniverse.fibers.Suspendable;
import io.kaleido.kat.states.KatOrderingContext;
import net.corda.core.contracts.ContractState;
import net.corda.core.crypto.SecureHash;
import net.corda.core.flows.*;
import net.corda.core.transactions.SignedTransaction;
import net.corda.core.utilities.ProgressTracker;
import static net.corda.core.contracts.ContractsDSL.requireThat;

@InitiatedBy(CreateOrderingContextFlow.class)
public class ReceiveOrderingContextFlow extends FlowLogic<SignedTransaction> {
    private final FlowSession otherPartySession;

    public ReceiveOrderingContextFlow(FlowSession otherPartySession) {
        this.otherPartySession = otherPartySession;
    }

    @Suspendable
    @Override
    public SignedTransaction call() throws FlowException {
        class SignTxFlow extends SignTransactionFlow {
            private SignTxFlow(FlowSession otherPartyFlow, ProgressTracker progressTracker) {
                super(otherPartyFlow, progressTracker);
            }

            @Override
            protected void checkTransaction(SignedTransaction stx) {
                requireThat(require -> {
                    ContractState output = stx.getTx().getOutputs().get(0).getData();
                    require.using("This must be an ordering context creation transaction.", output instanceof KatOrderingContext);
                    KatOrderingContext orderingContext = (KatOrderingContext) output;
                    return null;
                });
            }
        }
        final SignTxFlow signTxFlow = new SignTxFlow(otherPartySession, SignTransactionFlow.Companion.tracker());
        final SecureHash txId = subFlow(signTxFlow).getId();
        return subFlow(new ReceiveFinalityFlow(otherPartySession, txId));
    }
}
