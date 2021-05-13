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

package io.kaleido.kat.contracts;

import static net.corda.core.contracts.ContractsDSL.requireSingleCommand;
import static net.corda.core.contracts.ContractsDSL.requireThat;

import io.kaleido.kat.states.AssetEventState;
import io.kaleido.kat.states.KatOrderingContext;
import net.corda.core.contracts.CommandData;
import net.corda.core.contracts.CommandWithParties;
import net.corda.core.contracts.Contract;
import net.corda.core.identity.AbstractParty;
import net.corda.core.transactions.LedgerTransaction;

import java.security.PublicKey;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.stream.Collectors;

public class AssetTrailContract implements Contract {
    public static final String ID = "io.kaleido.kat.contracts.AssetTrailContract";

    @Override
    public void verify(LedgerTransaction tx) {
        final CommandWithParties<Commands> command = requireSingleCommand(tx.getCommands(), Commands.class);
        final Commands commandData = command.getValue();
        final Set<PublicKey> setOfSigners = new HashSet<>(command.getSigners());
        if(commandData instanceof Commands.AssetEventCreate) {
            verifyAssetEventCreate(tx, setOfSigners);
        } else if(commandData instanceof Commands.OrderingContextCreate) {
            verifyOrderingContextCreate(tx, setOfSigners);
        } else {
            throw new IllegalArgumentException("Unrecognised command.");
        }
    }

    private void verifyAssetEventCreate(LedgerTransaction tx, Set<PublicKey> signers) {
        requireThat(require -> {
            List<KatOrderingContext> inContexts = tx.inputsOfType(KatOrderingContext.class);
            List<KatOrderingContext> outContexts = tx.outputsOfType(KatOrderingContext.class);
            List<AssetEventState> eventStates = tx.outputsOfType(AssetEventState.class);
            require.using("An ordering context must be consumed when creating a Kat Event.",
                    tx.getInputs().size() == 1 && inContexts.size() == 1);
            require.using("One kat event and a new ordering context should be created.",
                    tx.getOutputs().size() == 2 && outContexts.size() == 1 && eventStates.size() == 1);
            final KatOrderingContext inContext = inContexts.get(0);
            final KatOrderingContext outContext = outContexts.get(0);
            final AssetEventState outState = eventStates.get(0);
            require.using("The nonce value must be incremented by 1.",
                    outContext.getNonce() == inContext.getNonce()+1);
            require.using("The linearId value must be same.",
                    outContext.getLinearId().equals(inContext.getLinearId()));
            require.using("participants of input context should be same as output context", inContext.getParticipants().equals(outContext.getParticipants()));
            require.using("author of output state must be same as author of output context",
                    outContext.getAuthor().getOwningKey().equals(outState.getAuthor().getOwningKey()));
            require.using("author must be a signer", signers.contains(outState.getAuthor().getOwningKey()));
            return null;
        });
    }

    private void verifyOrderingContextCreate(LedgerTransaction tx, Set<PublicKey> signers) {
        requireThat(require -> {
            List<KatOrderingContext> outContexts = tx.outputsOfType(KatOrderingContext.class);
            require.using("No inputs should be consumed when creating an ordering context between parties.",
                    tx.getInputs().isEmpty());
            require.using("Only one output state should be created.",
                    tx.getOutputs().size() == 1 && outContexts.size() == 1);
            final KatOrderingContext out = outContexts.get(0);
            final List<PublicKey> keys = out.getParticipants().stream().map(AbstractParty::getOwningKey).collect(Collectors.toList());
            require.using("All of the participants must be signers.",
                    signers.containsAll(keys));
            require.using("author should be one of the participants", keys.contains(out.getAuthor().getOwningKey()));
            require.using("The nonce value must be 0.",
                    out.getNonce() == 0);
            return null;
        });
    }

    public interface Commands extends CommandData {
        class AssetEventCreate implements Commands {}
        class OrderingContextCreate implements Commands {}
    }
}
