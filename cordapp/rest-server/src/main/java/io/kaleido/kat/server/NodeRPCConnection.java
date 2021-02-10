package io.kaleido.kat.server;

import net.corda.client.rpc.CordaRPCClient;
import net.corda.client.rpc.CordaRPCConnection;
import net.corda.core.messaging.CordaRPCOps;
import net.corda.core.utilities.NetworkHostAndPort;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import javax.annotation.PostConstruct;
import javax.annotation.PreDestroy;

/**
 * Wraps a node RPC proxy.
 */
@Component
public class NodeRPCConnection implements AutoCloseable {
    private final String host;
    private final String username;
    private final String password;
    private final int rpcPort;

    private CordaRPCConnection rpcConnection;
    private CordaRPCOps proxy; // The RPC proxy

    /**
     * The RPC proxy is configured based on the properties in `application.properties`.
     * @param host The host of the node we are connecting to.
     * @param rpcPort The RPC port of the node we are connecting to.
     * @param username The username for logging into the RPC client.
     * @param password The password for logging into the RPC client.
     */
    public NodeRPCConnection(
            @Value("${rpc.host}") String host,
            @Value("${rpc.username}") String username,
            @Value("${rpc.password}") String password,
            @Value("${rpc.port}") int rpcPort
    ) {
        this.host = host;
        this.username = username;
        this.password = password;
        this.rpcPort = rpcPort;
    }

    @PostConstruct
    public void initialiseNodeRPCConnection() {
        NetworkHostAndPort rpcAddress = new NetworkHostAndPort(host, rpcPort);
        CordaRPCClient rpcClient = new CordaRPCClient(rpcAddress);
        this.rpcConnection = rpcClient.start(username, password);
        this.proxy = rpcConnection.getProxy();
    }

    public CordaRPCOps getProxy() {
        return proxy;
    }

    @PreDestroy
    @Override
    public void close() throws Exception {
        rpcConnection.notifyServerAndClose();
    }
}
