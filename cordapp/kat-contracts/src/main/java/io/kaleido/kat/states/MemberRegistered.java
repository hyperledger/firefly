package io.kaleido.kat.states;

import io.kaleido.kat.contracts.KatContract;
import net.corda.core.contracts.BelongsToContract;
import net.corda.core.contracts.ContractState;
import net.corda.core.identity.AbstractParty;
import net.corda.core.identity.Party;
import org.jetbrains.annotations.NotNull;

import java.util.List;

@BelongsToContract(KatContract.class)
public class MemberRegistered implements ContractState {
    private final Party member;
    private final String name;
    private final String assetTrailInstanceID;
    private final String app2appDestination;
    private final String docExchangeDestination;

    public MemberRegistered(Party member, String name, String assetTrailInstanceID, String app2appDestination, String docExchangeDestination) {
        this.member = member;
        this.name = name;
        this.assetTrailInstanceID = assetTrailInstanceID;
        this.app2appDestination = app2appDestination;
        this.docExchangeDestination = docExchangeDestination;
    }

    @NotNull
    @Override
    public List<AbstractParty> getParticipants() {
        return List.of(member);
    }

    @Override
    public String toString() {
        return String.format("MemberRegistered(member=%s, name=%s, assetTrailInstanceID=%s, app2appDestination=%s, docExchangeDestination=%s)", member, name, assetTrailInstanceID, app2appDestination, docExchangeDestination);
    }
}
