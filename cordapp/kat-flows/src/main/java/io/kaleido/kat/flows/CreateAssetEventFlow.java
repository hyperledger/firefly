package io.kaleido.kat.flows;

import com.google.common.collect.ImmutableList;
import io.kaleido.kat.contracts.KatContract;
import io.kaleido.kat.states.KatOrderingContext;
import net.corda.core.contracts.Command;
import net.corda.core.contracts.ContractState;
import net.corda.core.contracts.StateAndRef;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.FinalityFlow;
import net.corda.core.flows.FlowException;
import net.corda.core.flows.FlowLogic;
import net.corda.core.identity.Party;
import net.corda.core.node.services.Vault;
import net.corda.core.node.services.vault.QueryCriteria;
import net.corda.core.transactions.SignedTransaction;
import net.corda.core.transactions.TransactionBuilder;
import net.corda.core.utilities.ProgressTracker;
import java.util.Collections;
import java.util.HashSet;
import java.util.List;

public class CreateAssetEventFlow<T extends ContractState> extends FlowLogic<SignedTransaction> {
    private final List<Party> observers;
    private final UniqueIdentifier orderingContext;
    private final ProgressTracker.Step GENERATING_TRANSACTION = new ProgressTracker.Step("Generating transaction based on new AssetInstanceBatchCreated.");
    private final ProgressTracker.Step VERIFYING_TRANSACTION = new ProgressTracker.Step("Verifying contract constraints.");
    private final ProgressTracker.Step SIGNING_TRANSACTION = new ProgressTracker.Step("Signing transaction with our private key.");
    private final ProgressTracker.Step FINALISING_TRANSACTION = new ProgressTracker.Step("Obtaining notary signature and recording transaction.") {
        @Override
        public ProgressTracker childProgressTracker() {
            return FinalityFlow.Companion.tracker();
        }
    };
    private final ProgressTracker.Step SENDING_TRANSACTION_TO_OBSERVERS = new ProgressTracker.Step("Sending final transaction to observers");

    // The progress tracker checkpoints each stage of the flow and outputs the specified messages when each
    // checkpoint is reached in the code. See the 'progressTracker.currentStep' expressions within the call()
    // function.
    private final ProgressTracker progressTracker = new ProgressTracker(
            GENERATING_TRANSACTION,
            VERIFYING_TRANSACTION,
            SIGNING_TRANSACTION,
            FINALISING_TRANSACTION
    );

    public T getAssetEvent(){
        return null;
    }

    public KatOrderingContext updateOrderingContext(KatOrderingContext oldContext) {
        return new KatOrderingContext(oldContext.getLinearId(), new HashSet(oldContext.getParticipants()), oldContext.getNonce()+1);
    }

    public CreateAssetEventFlow(List<Party> observers, UniqueIdentifier orderingContext) {
        this.observers = observers;
        this.orderingContext = orderingContext;
    }

    private StateAndRef<KatOrderingContext> getOrderingContext(UniqueIdentifier linearId) throws FlowException {
        QueryCriteria queryCriteria = new QueryCriteria.LinearStateQueryCriteria(
                null,
                ImmutableList.of(linearId),
                Vault.StateStatus.UNCONSUMED,
                null);

        List<StateAndRef<KatOrderingContext>> existingContexts = getServiceHub().getVaultService().queryBy(KatOrderingContext.class, queryCriteria).getStates();
        if (existingContexts.size() != 1) {
            throw new FlowException(String.format("ordering context with id %s not found.", linearId));
        }
        return existingContexts.get(0);
    }

    @Override
    public ProgressTracker getProgressTracker() {
        return progressTracker;
    }

    @Override
    public SignedTransaction call() throws FlowException {
        // Obtain a reference to the notary we want to use.
        final Party notary = getServiceHub().getNetworkMapCache().getNotaryIdentities().get(0);
        // Generate an unsigned transaction.
        progressTracker.setCurrentStep(GENERATING_TRANSACTION);
        Party me = getOurIdentity();
        final Command<KatContract.Commands.AssetEventCreate> txCommand = new Command<>(
                new KatContract.Commands.AssetEventCreate(),
                ImmutableList.of(me.getOwningKey()));
        final StateAndRef<KatOrderingContext> inContext = getOrderingContext(orderingContext);
        final T output = getAssetEvent();
        final TransactionBuilder txBuilder = new TransactionBuilder(notary)
                .addInputState(inContext)
                .addOutputState(output, KatContract.ID)
                .addOutputState(updateOrderingContext(inContext.getState().getData()), KatContract.ID)
                .addCommand(txCommand);
        progressTracker.setCurrentStep(VERIFYING_TRANSACTION);
        txBuilder.verify(getServiceHub());

        progressTracker.setCurrentStep(SIGNING_TRANSACTION);
        final SignedTransaction signedTx = getServiceHub().signInitialTransaction(txBuilder);

        progressTracker.setCurrentStep(FINALISING_TRANSACTION);
        final SignedTransaction finalTx = subFlow(new FinalityFlow(signedTx, Collections.emptyList()));

        progressTracker.setCurrentStep(SENDING_TRANSACTION_TO_OBSERVERS);
        for(Party observer: observers) {
            subFlow(new SendTxToObserverNodeFlow(observer, finalTx));
        }
        return finalTx;
    }
}
