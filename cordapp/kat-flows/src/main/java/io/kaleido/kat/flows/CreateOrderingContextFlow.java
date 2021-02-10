package io.kaleido.kat.flows;
import co.paralleluniverse.fibers.Suspendable;
import io.kaleido.kat.contracts.KatContract;
import io.kaleido.kat.states.KatOrderingContext;
import net.corda.core.contracts.Command;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.*;
import net.corda.core.identity.AbstractParty;
import net.corda.core.identity.Party;
import net.corda.core.transactions.SignedTransaction;
import net.corda.core.transactions.TransactionBuilder;
import net.corda.core.utilities.ProgressTracker;

import java.security.PublicKey;
import java.util.List;
import java.util.Set;
import java.util.stream.Collectors;

@InitiatingFlow
@StartableByRPC
public class CreateOrderingContextFlow extends FlowLogic<SignedTransaction> {
    private final UniqueIdentifier contextId;
    private final Set<AbstractParty> partiesForContext;
    private final ProgressTracker.Step GENERATING_TRANSACTION = new ProgressTracker.Step("Generating transaction based on new AssetInstanceBatchCreated.");
    private final ProgressTracker.Step VERIFYING_TRANSACTION = new ProgressTracker.Step("Verifying contract constraints.");
    private final ProgressTracker.Step SIGNING_TRANSACTION = new ProgressTracker.Step("Signing transaction with our private key.");
    private final ProgressTracker.Step COLLECTING_SIGNATURES = new ProgressTracker.Step("Collecting signatures from parties within ordering context.") {
        @Override
        public ProgressTracker childProgressTracker() {
            return CollectSignaturesFlow.Companion.tracker();
        }
    };
    private final ProgressTracker.Step FINALISING_TRANSACTION = new ProgressTracker.Step("Obtaining notary signature and recording transaction.") {
        @Override
        public ProgressTracker childProgressTracker() {
            return FinalityFlow.Companion.tracker();
        }
    };

    // The progress tracker checkpoints each stage of the flow and outputs the specified messages when each
    // checkpoint is reached in the code. See the 'progressTracker.currentStep' expressions within the call()
    // function.
    private final ProgressTracker progressTracker = new ProgressTracker(
            GENERATING_TRANSACTION,
            VERIFYING_TRANSACTION,
            SIGNING_TRANSACTION,
            COLLECTING_SIGNATURES,
            FINALISING_TRANSACTION
    );

    public CreateOrderingContextFlow(UniqueIdentifier contextId, Set<AbstractParty> partiesForContext) {
        this.contextId = contextId;
        this.partiesForContext = partiesForContext;
    }

    @Suspendable
    @Override
    public SignedTransaction call() throws FlowException {
        // Obtain a reference to the notary we want to use.
        final Party notary = getServiceHub().getNetworkMapCache().getNotaryIdentities().get(0);
        // Generate an unsigned transaction.
        progressTracker.setCurrentStep(GENERATING_TRANSACTION);
        final List<PublicKey> signers = partiesForContext.stream().map(AbstractParty::getOwningKey).collect(Collectors.toList());
        final Command<KatContract.Commands.OrderingContextCreate> txCommand = new Command<>(
                new KatContract.Commands.OrderingContextCreate(),
                signers);
        final KatOrderingContext newContext = new KatOrderingContext(contextId, partiesForContext, 0);

        final TransactionBuilder txBuilder = new TransactionBuilder(notary)
                .addOutputState(newContext, KatContract.ID)
                .addCommand(txCommand);
        progressTracker.setCurrentStep(VERIFYING_TRANSACTION);
        txBuilder.verify(getServiceHub());

        progressTracker.setCurrentStep(SIGNING_TRANSACTION);
        final SignedTransaction signedTx = getServiceHub().signInitialTransaction(txBuilder);

        progressTracker.setCurrentStep(COLLECTING_SIGNATURES);
        Set<FlowSession> flowSessions = partiesForContext.stream().filter(party -> !party.getOwningKey().equals(getOurIdentity().getOwningKey())).map(this::initiateFlow).collect(Collectors.toSet());
        SignedTransaction fullySignedTx = subFlow(new CollectSignaturesFlow(signedTx, flowSessions, COLLECTING_SIGNATURES.childProgressTracker()));
        progressTracker.setCurrentStep(FINALISING_TRANSACTION);
        return subFlow(new FinalityFlow(fullySignedTx, flowSessions, FINALISING_TRANSACTION.childProgressTracker()));
    }
}
