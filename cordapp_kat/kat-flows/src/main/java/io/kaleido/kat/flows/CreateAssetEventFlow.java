// Copyright © 2021 Kaleido, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package io.kaleido.kat.flows;

import co.paralleluniverse.fibers.Suspendable;
import com.google.common.collect.ImmutableList;
import io.kaleido.kat.contracts.AssetTrailContract;
import io.kaleido.kat.states.KatOrderingContext;
import net.corda.core.contracts.Command;
import net.corda.core.contracts.ContractState;
import net.corda.core.contracts.StateAndRef;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.FinalityFlow;
import net.corda.core.flows.FlowException;
import net.corda.core.flows.FlowLogic;
import net.corda.core.flows.FlowSession;
import net.corda.core.identity.AbstractParty;
import net.corda.core.identity.Party;
import net.corda.core.node.services.Vault;
import net.corda.core.node.services.vault.PageSpecification;
import net.corda.core.node.services.vault.QueryCriteria;
import net.corda.core.transactions.SignedTransaction;
import net.corda.core.transactions.TransactionBuilder;
import net.corda.core.utilities.ProgressTracker;

import java.nio.ByteBuffer;
import java.util.*;
import java.util.stream.Collectors;

import static net.corda.core.node.services.vault.QueryCriteriaUtils.DEFAULT_PAGE_NUM;
import static net.corda.core.node.services.vault.QueryCriteriaUtils.DEFAULT_PAGE_SIZE;

public class CreateAssetEventFlow<T extends ContractState> extends FlowLogic<SignedTransaction> {
    protected final List<Party> observers;
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

    public T getAssetEvent(){
        return null;
    }

    public KatOrderingContext updateOrderingContext(KatOrderingContext oldContext) {
        return new KatOrderingContext(oldContext.getLinearId(), getOurIdentity(), new HashSet<>(oldContext.getParticipants()), oldContext.getNonce()+1);
    }

    public CreateAssetEventFlow(List<Party> observers) {
        this.observers = observers;
    }

    @Suspendable
    private StateAndRef<KatOrderingContext> getOrderingContext() throws FlowException {
        Set<AbstractParty> partiesInContext = new HashSet<>(this.observers);
        partiesInContext.add(getOurIdentity());
        UniqueIdentifier linearId = new UniqueIdentifier(null, UUID.nameUUIDFromBytes(ByteBuffer.allocate(4).putInt(partiesInContext.hashCode()).array()));
        QueryCriteria queryCriteria = new QueryCriteria.LinearStateQueryCriteria(
                new ArrayList<>(partiesInContext),
                ImmutableList.of(linearId),
                Vault.StateStatus.UNCONSUMED,
                null);

        List<StateAndRef<KatOrderingContext>> existingContexts = new ArrayList<>();
        int currentPage = DEFAULT_PAGE_NUM;
        while(true) {
            Vault.Page<KatOrderingContext> res = getServiceHub().getVaultService().queryBy(KatOrderingContext.class, queryCriteria, new PageSpecification(currentPage, DEFAULT_PAGE_SIZE));
            if(res.getStates().isEmpty()) {
                break;
            } else {
                existingContexts.addAll(res.getStates());
                currentPage++;
            }
        }
        if (existingContexts.size() == 0) {
            return subFlow(new CreateOrderingContextFlow(linearId, partiesInContext));
        } else if(existingContexts.size() > 1) {
            // Break Ties With nonce's first
            // If same nonce use author's PublicKey string natural order
            Comparator<StateAndRef<KatOrderingContext>> sortByNonce = Comparator.comparingLong((StateAndRef<KatOrderingContext> o) -> o.getState().getData().getNonce());
            Comparator<StateAndRef<KatOrderingContext>> sortByAuthor = Comparator.comparing((StateAndRef<KatOrderingContext> o) -> o.getState().getData().getAuthor().getOwningKey().toString());
            Collections.sort(existingContexts, sortByNonce.reversed());
            Collections.sort(existingContexts, sortByAuthor);
        }
        return existingContexts.get(0);
    }

    @Override
    public ProgressTracker getProgressTracker() {
        return progressTracker;
    }

    @Suspendable
    @Override
    public SignedTransaction call() throws FlowException {
        // Obtain a reference to the notary we want to use.
        final Party notary = getServiceHub().getNetworkMapCache().getNotaryIdentities().get(0);
        // Generate an unsigned transaction.
        progressTracker.setCurrentStep(GENERATING_TRANSACTION);
        Party me = getOurIdentity();
        final Command<AssetTrailContract.Commands.AssetEventCreate> txCommand = new Command<>(
                new AssetTrailContract.Commands.AssetEventCreate(),
                ImmutableList.of(me.getOwningKey()));
        final StateAndRef<KatOrderingContext> inContext = getOrderingContext();
        final T output = getAssetEvent();
        final TransactionBuilder txBuilder = new TransactionBuilder(notary)
                .addInputState(inContext)
                .addOutputState(output, AssetTrailContract.ID)
                .addOutputState(updateOrderingContext(inContext.getState().getData()), AssetTrailContract.ID)
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
