package io.kaleido.kat.flows;

import io.kaleido.kat.states.AssetInstanceBatchCreated;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.flows.*;
import net.corda.core.identity.Party;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.ArrayList;
import java.util.List;

@InitiatingFlow
@StartableByRPC
public class CreateAssetInstanceBatchFlow extends CreateAssetEventFlow<AssetInstanceBatchCreated> {
    private final String batchHash;
    public CreateAssetInstanceBatchFlow(String batchHash, List<Party> observers) {
        super(observers);
        this.batchHash = batchHash;
    }

    @Override
    public AssetInstanceBatchCreated getAssetEvent(){
        List<Party> participants = new ArrayList<>(this.observers);
        participants.add(getOurIdentity());
        return new AssetInstanceBatchCreated(getOurIdentity(), batchHash, participants);
    }
}
