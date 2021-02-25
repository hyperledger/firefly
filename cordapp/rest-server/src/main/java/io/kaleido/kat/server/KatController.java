package io.kaleido.kat.server;

import com.fasterxml.jackson.databind.ObjectMapper;
import io.kaleido.kat.flows.*;
import io.kaleido.kat.server.data.*;
import io.kaleido.kat.states.KatOrderingContext;
import net.corda.client.jackson.JacksonSupport;
import net.corda.core.contracts.UniqueIdentifier;
import net.corda.core.identity.AbstractParty;
import net.corda.core.identity.CordaX500Name;
import net.corda.core.identity.Party;
import net.corda.core.messaging.CordaRPCOps;
import net.corda.core.node.services.Vault;
import net.corda.core.node.services.vault.QueryCriteria;
import net.corda.core.transactions.SignedTransaction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.server.ResponseStatusException;

import java.util.*;
import java.util.concurrent.ExecutionException;
import java.util.stream.Collectors;

import static org.springframework.http.MediaType.APPLICATION_JSON_VALUE;

@RestController
@RequestMapping("api/v1/kat")
public class KatController {
    private static final Logger logger = LoggerFactory.getLogger(RestController.class);
    private final CordaRPCOps proxy;
    private final CordaX500Name me;

    public KatController(NodeRPCConnection rpc) {
        this.proxy = rpc.getProxy();
        this.me = proxy.nodeInfo().getLegalIdentities().get(0).getName();

    }

    private QueryCriteria.VaultQueryCriteria addExactParticipants(QueryCriteria.VaultQueryCriteria criteria, List<AbstractParty> exactParticipants) {
       return new QueryCriteria.VaultQueryCriteria(
               criteria.getStatus(),
               criteria.getContractStateTypes(),
               criteria.getStateRefs(),
               criteria.getNotary(),
               criteria.getSoftLockingCondition(),
               criteria.getTimeCondition(),
               criteria.getRelevancyStatus(),
               criteria.getConstraintTypes(),
               criteria.getConstraints(),
               criteria.getParticipants(),
               criteria.getExternalIds(),
               exactParticipants
       );
    }

    private UniqueIdentifier createOrGetOrderingContext(List<AbstractParty> partiesInContext) {
        UniqueIdentifier contextId;
        QueryCriteria.VaultQueryCriteria _criteria = new QueryCriteria.VaultQueryCriteria()
                .withStatus(Vault.StateStatus.UNCONSUMED)
                .withContractStateTypes(Set.of(KatOrderingContext.class));
        QueryCriteria.VaultQueryCriteria criteria = addExactParticipants(_criteria, partiesInContext);
        Vault.Page<KatOrderingContext> res = proxy.vaultQueryByCriteria(criteria, KatOrderingContext.class);
        if(res.getStates().isEmpty()) {
            // create ordering context if doesn't exist
            logger.info("Creating new ordering context for parties {}", partiesInContext);
            UniqueIdentifier uuid = new UniqueIdentifier(null, UUID.randomUUID());
            try {
                proxy.startFlowDynamic(CreateOrderingContextFlow.class, uuid, new HashSet<>(partiesInContext)).getReturnValue().get();
                contextId = uuid;
            } catch (InterruptedException | ExecutionException e) {
                logger.error("failed to create the ordering context", e);
                throw new ResponseStatusException(HttpStatus.INTERNAL_SERVER_ERROR, "failed to create ordering context", e);
            }
        } else {
            contextId = res.getStates().get(0).getState().getData().getLinearId();
        }
        logger.info("Ordering context for parties {}, with id {}", partiesInContext, contextId);
        return contextId;
    }

    @Configuration
    class Plugin {
        @Bean
        public ObjectMapper registerModule() {
            return JacksonSupport.createNonRpcMapper();
        }
    }

    @GetMapping(value = "/identities", produces = APPLICATION_JSON_VALUE)
    public List<String> identities() {
        return proxy.nodeInfo().getLegalIdentities().stream().map(Party::toString).collect(Collectors.toList());
    }

    @PostMapping(value = "/createAssetInstance", consumes = APPLICATION_JSON_VALUE, produces = APPLICATION_JSON_VALUE)
    public AssetResponse createAssetInstance(@RequestBody CreateAssetInstanceRequest request) {
        // get ordering context
        List<AbstractParty> participantList = new ArrayList<>();
        for(String observer: request.getParticipants()) {
            Party party = proxy.wellKnownPartyFromX500Name(CordaX500Name.parse(observer));
            if(party == null) {
                logger.error("No party with name {} found", observer);
                throw new ResponseStatusException(HttpStatus.BAD_REQUEST, "No party with name "+observer+" found.");
            }
            participantList.add(party);
        }
        List<AbstractParty> partiesInContext = new ArrayList<>(participantList);
        partiesInContext.add(proxy.nodeInfo().getLegalIdentities().get(0));
        UniqueIdentifier contextId = createOrGetOrderingContext(partiesInContext);
        AssetResponse response = new AssetResponse();
        try {
            SignedTransaction assetTxResult = proxy.startFlowDynamic(CreateAssetInstanceFlow.class, request.getAssetInstanceID(), request.getAssetDefinitionID(), request.getContentHash(), participantList, contextId).getReturnValue().get();
            response.setTxHash(assetTxResult.getId().toString());
            return response;
        } catch (InterruptedException | ExecutionException e) {
            logger.error("failed to create assetInstance", e);
            throw new ResponseStatusException(HttpStatus.INTERNAL_SERVER_ERROR, "failed to create assetInstance", e);
        }
    }

    @PostMapping(value = "/createAssetInstanceBatch", consumes = APPLICATION_JSON_VALUE, produces = APPLICATION_JSON_VALUE)
    public AssetResponse createAssetInstanceBatch(@RequestBody CreateAssetInstanceBatchRequest request) {
        // get ordering context
        List<AbstractParty> participantList = new ArrayList<>();
        for(String observer: request.getParticipants()) {
            Party party = proxy.wellKnownPartyFromX500Name(CordaX500Name.parse(observer));
            if(party == null) {
                logger.error("No party with name {} found", observer);
                throw new ResponseStatusException(HttpStatus.BAD_REQUEST, "No party with name "+observer+" found.");
            }
            participantList.add(party);
        }
        List<AbstractParty> partiesInContext = new ArrayList<>(participantList);
        partiesInContext.add(proxy.nodeInfo().getLegalIdentities().get(0));
        UniqueIdentifier contextId = createOrGetOrderingContext(partiesInContext);
        AssetResponse response = new AssetResponse();
        try {
            SignedTransaction assetTxResult = proxy.startFlowDynamic(CreateAssetInstanceBatchFlow.class, request.getBatchHash(), participantList, contextId).getReturnValue().get();
            response.setTxHash(assetTxResult.getId().toString());
            return response;
        } catch (InterruptedException | ExecutionException e) {
            logger.error("failed to create assetInstanceBatch", e);
            throw new ResponseStatusException(HttpStatus.INTERNAL_SERVER_ERROR, "failed to create assetInstanceBatch", e);
        }
    }

    @PostMapping(value = "/createDescribedAssetInstance", consumes = APPLICATION_JSON_VALUE, produces = APPLICATION_JSON_VALUE)
    public AssetResponse createDescribedAssetInstance(@RequestBody CreateDescribedAssetInstanceRequest request) {
        // get ordering context
        List<AbstractParty> participantList = new ArrayList<>();
        for(String observer: request.getParticipants()) {
            Party party = proxy.wellKnownPartyFromX500Name(CordaX500Name.parse(observer));
            if(party == null) {
                logger.error("No party with name {} found", observer);
                throw new ResponseStatusException(HttpStatus.BAD_REQUEST, "No party with name "+observer+" found.");
            }
            participantList.add(party);
        }
        List<AbstractParty> partiesInContext = new ArrayList<>(participantList);
        partiesInContext.add(proxy.nodeInfo().getLegalIdentities().get(0));
        UniqueIdentifier contextId = createOrGetOrderingContext(partiesInContext);
        AssetResponse response = new AssetResponse();
        try {
            SignedTransaction assetTxResult = proxy.startFlowDynamic(CreateDescribedAssetInstanceFlow.class, request.getAssetInstanceID(), request.getAssetDefinitionID(), request.getDescriptionHash(), request.getContentHash(), participantList, contextId).getReturnValue().get();
            response.setTxHash(assetTxResult.getId().toString());
            return response;
        } catch (InterruptedException | ExecutionException e) {
            logger.error("failed to create describedAssetInstance", e);
            throw new ResponseStatusException(HttpStatus.INTERNAL_SERVER_ERROR, "failed to create describedAssetInstance", e);
        }
    }

    @PostMapping(value = "/setAssetInstanceProperty", consumes = APPLICATION_JSON_VALUE, produces = APPLICATION_JSON_VALUE)
    public AssetResponse setAssetInstanceProperty(@RequestBody SetAssetInstancePropertyRequest request) {
        // get ordering context
        List<AbstractParty> participantList = new ArrayList<>();
        for(String observer: request.getParticipants()) {
            Party party = proxy.wellKnownPartyFromX500Name(CordaX500Name.parse(observer));
            if(party == null) {
                logger.error("No party with name {} found", observer);
                throw new ResponseStatusException(HttpStatus.BAD_REQUEST, "No party with name "+observer+" found.");
            }
            participantList.add(party);
        }
        List<AbstractParty> partiesInContext = new ArrayList<>(participantList);
        partiesInContext.add(proxy.nodeInfo().getLegalIdentities().get(0));
        UniqueIdentifier contextId = createOrGetOrderingContext(partiesInContext);
        AssetResponse response = new AssetResponse();
        try {
            SignedTransaction assetTxResult = proxy.startFlowDynamic(SetAssetInstancePropertyFlow.class, request.getAssetDefinitionID(), request.getAssetInstanceID(), request.getKey(), request.getValue(), participantList, contextId).getReturnValue().get();
            response.setTxHash(assetTxResult.getId().toString());
            return response;
        } catch (InterruptedException | ExecutionException e) {
            logger.error("failed to create unstructuredAssetDefinition", e);
            throw new ResponseStatusException(HttpStatus.INTERNAL_SERVER_ERROR, "failed to create unstructuredAssetDefinition", e);
        }
    }
}
