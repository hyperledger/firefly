package io.kaleido.kat.flows;

import co.paralleluniverse.fibers.Suspendable;
import com.google.common.collect.ImmutableList;
import io.kaleido.kat.contracts.AssetTrailContract;
import io.kaleido.kat.states.MemberRegistered;
import net.corda.core.contracts.Command;
import net.corda.core.flows.*;
import net.corda.core.identity.Party;
import net.corda.core.transactions.SignedTransaction;
import net.corda.core.transactions.TransactionBuilder;
import net.corda.core.utilities.ProgressTracker;

import java.util.List;
import java.util.Set;
import java.util.stream.Collectors;

@InitiatingFlow
@StartableByRPC
public class RegisterMemberFlow extends FlowLogic<SignedTransaction> {
    private final String name;
    private final String assetTrailInstanceID;
    private final String app2appDestination;
    private final String docExchangeDestination;
    private final List<Party> observers;
    private final ProgressTracker.Step GENERATING_TRANSACTION = new ProgressTracker.Step("Generating transaction based on new AssetInstanceBatchCreated.");
    private final ProgressTracker.Step VERIFYING_TRANSACTION = new ProgressTracker.Step("Verifying contract constraints.");
    private final ProgressTracker.Step SIGNING_TRANSACTION = new ProgressTracker.Step("Signing transaction with our private key.");
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
            FINALISING_TRANSACTION
    );

    public RegisterMemberFlow(String name, String assetTrailInstanceID, String app2appDestination, String docExchangeDestination, List<Party> observers) {
        this.name = name;
        this.assetTrailInstanceID = assetTrailInstanceID;
        this.app2appDestination = app2appDestination;
        this.docExchangeDestination = docExchangeDestination;
        this.observers = observers;
    }

    @Suspendable
    @Override
    public SignedTransaction call() throws FlowException {
        // Obtain a reference to the notary we want to use.
        final Party notary = getServiceHub().getNetworkMapCache().getNotaryIdentities().get(0);
        // Generate an unsigned transaction.
        progressTracker.setCurrentStep(GENERATING_TRANSACTION);
        Party me = getOurIdentity();
        final Command<AssetTrailContract.Commands.MemberCreate> txCommand = new Command<>(
                new AssetTrailContract.Commands.MemberCreate(),
                ImmutableList.of(me.getOwningKey()));
        final MemberRegistered output = new MemberRegistered(getOurIdentity(), name, assetTrailInstanceID, app2appDestination, docExchangeDestination);
        final TransactionBuilder txBuilder = new TransactionBuilder(notary)
                .addOutputState(output, AssetTrailContract.ID)
                .addCommand(txCommand);
        progressTracker.setCurrentStep(VERIFYING_TRANSACTION);
        txBuilder.verify(getServiceHub());

        progressTracker.setCurrentStep(SIGNING_TRANSACTION);
        final SignedTransaction signedTx = getServiceHub().signInitialTransaction(txBuilder);

        progressTracker.setCurrentStep(FINALISING_TRANSACTION);
        Set<FlowSession> flowSessions = observers.stream().map(this::initiateFlow).collect(Collectors.toSet());
        return subFlow(new FinalityFlow(signedTx, flowSessions));
    }
}
