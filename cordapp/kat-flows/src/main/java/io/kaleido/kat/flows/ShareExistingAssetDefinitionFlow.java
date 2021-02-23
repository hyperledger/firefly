package io.kaleido.kat.flows;

import co.paralleluniverse.fibers.Suspendable;
import io.kaleido.kat.states.AssetDefinitionCreated;
import net.corda.core.crypto.SecureHash;
import net.corda.core.flows.*;
import net.corda.core.identity.Party;
import net.corda.core.transactions.SignedTransaction;
import net.corda.core.utilities.ProgressTracker;

import java.util.List;
import java.util.Set;
import java.util.stream.Collectors;

@InitiatingFlow
@StartableByRPC
public class ShareExistingAssetDefinitionFlow extends FlowLogic<SignedTransaction> {
    private final List<Party> observers;
    private final String transactionHash;
    private final ProgressTracker.Step FINDING_TRANSACTION = new ProgressTracker.Step("Finding transaction based on transactionHash.");
    private final ProgressTracker.Step SHARING_TRANSACTION = new ProgressTracker.Step("Verifying contract constraints.");

    public ShareExistingAssetDefinitionFlow(String transactionHash, List<Party> observers) {
        this.observers = observers;
        this.transactionHash = transactionHash;
    }

    // The progress tracker checkpoints each stage of the flow and outputs the specified messages when each
    // checkpoint is reached in the code. See the 'progressTracker.currentStep' expressions within the call()
    // function.
    private final ProgressTracker progressTracker = new ProgressTracker(
            FINDING_TRANSACTION,
            SHARING_TRANSACTION
    );

    @Suspendable
    @Override
    public SignedTransaction call() throws FlowException {
        progressTracker.setCurrentStep(FINDING_TRANSACTION);
        final SignedTransaction signedTx = getServiceHub().getValidatedTransactions().getTransaction(SecureHash.parse(transactionHash));
        if(signedTx == null) {
            throw new FlowException(String.format("No transaction with hash %s was found", transactionHash));
        }
        if(signedTx.getCoreTransaction().outputsOfType(AssetDefinitionCreated.class).size() == 0) {
            throw new FlowException(String.format("Transaction with hash %s doesn't have asset definition as outputs", transactionHash));
        }
        progressTracker.setCurrentStep(SHARING_TRANSACTION);
        Set<FlowSession> flowSessions = observers.stream().map(this::initiateFlow).collect(Collectors.toSet());
        subFlow(new FinalityFlow(signedTx, flowSessions));
        return signedTx;
    }
}
