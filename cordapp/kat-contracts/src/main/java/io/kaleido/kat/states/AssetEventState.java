package io.kaleido.kat.states;
import net.corda.core.contracts.ContractState;
import net.corda.core.identity.Party;

import java.util.List;

public interface AssetEventState extends ContractState {
    Party getAuthor();
    List<Party> getAssetParticipants();
}

