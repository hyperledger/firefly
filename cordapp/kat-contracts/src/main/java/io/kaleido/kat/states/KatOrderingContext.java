package io.kaleido.kat.states;

import io.kaleido.kat.contracts.AssetTrailContract;
import net.corda.core.contracts.BelongsToContract;
import net.corda.core.contracts.LinearState;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.identity.AbstractParty;
import org.jetbrains.annotations.NotNull;

import java.util.ArrayList;
import java.util.List;
import java.util.Set;

@BelongsToContract(AssetTrailContract.class)
public class KatOrderingContext implements LinearState {
    private final UniqueIdentifier contextId;
    private final Set<AbstractParty> partiesForContext;
    private final long nonce;

    public KatOrderingContext(UniqueIdentifier contextId, Set<AbstractParty> partiesForContext, long nonce) {
        this.contextId = contextId;
        this.partiesForContext = partiesForContext;
        this.nonce = nonce;
    }

    public long getNonce() {
        return this.nonce;
    }

    @NotNull
    @Override
    public UniqueIdentifier getLinearId() {
        return contextId;
    }

    @NotNull
    @Override
    public List<AbstractParty> getParticipants() {
        return new ArrayList<>(partiesForContext);
    }
}
