package io.kaleido.kat.server;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jdk8.Jdk8Module;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import io.kaleido.kat.server.data.AssetResponse;
import io.kaleido.kat.server.data.CreateAssetInstanceRequest;
import net.corda.client.jackson.JacksonSupport;
import org.junit.AfterClass;
import org.junit.BeforeClass;
import org.junit.FixMethodOrder;
import org.junit.Test;
import org.junit.runners.MethodSorters;

import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.List;
import java.util.UUID;

import static java.lang.Thread.sleep;
import static org.junit.Assert.assertEquals;
@FixMethodOrder(MethodSorters.NAME_ASCENDING)
public class ComponentTests {
    /*
    @BeforeClass
    public static void startServer() throws InterruptedException {
        int retries = 0;
        int maxRetries = 15;
        long waitBeforeRetry = 1000;
        NodeRPCConnection connection = new NodeRPCConnection("localhost", "user1", "test", 10011);
        while (retries < maxRetries) {
            try {
                connection.initialiseNodeRPCConnection();
                connection.getProxy().nodeInfo();
                Server.main(new String[]{""});
            } catch (Exception e) {
                retries++;
                waitBeforeRetry *= 2;
                sleep(waitBeforeRetry);
                System.out.println("Connection, Retrying");
                continue;
            }
        }
    }

    @AfterClass
    public static void stopServer() {
        Server.stop();
    }

    private class HTTPResponse <T>  {
        T val;
        int status;

        public T getVal() {
            return val;
        }

        public void setVal(T val) {
            this.val = val;
        }

        public int getStatus() {
            return this.status;
        }

        public void setStatus(int status) {
            this.status = status;
        }

    }

    public <T> HTTPResponse<T>  request(String method, String path, Object request, Class<T> valueType ) throws Exception {
        // using rpcmapper here instead of nonRpcMapper to resolve parties from test network
        ObjectMapper objectMapper = JacksonSupport.createNonRpcMapper();
        objectMapper.registerModule(new JavaTimeModule());
        objectMapper.registerModule(new Jdk8Module());
        HttpURLConnection conn = (HttpURLConnection) new URL("http://localhost:9991" + path).openConnection();
        conn.setRequestMethod(method);
        if (request != null) {
            conn.setDoOutput(true);
            conn.setRequestProperty("Content-Type", "application/json");
        }
        conn.connect();
        HTTPResponse<T> res = new HTTPResponse<T>();
        try {
            if (request != null) {
                try (OutputStream os = conn.getOutputStream()) {
                    objectMapper.writeValue(os, request);
                }
            }
            res.setStatus(conn.getResponseCode());
            if (res.getStatus() >= HttpURLConnection.HTTP_BAD_REQUEST) {
                System.out.println(res.getStatus());
                try (InputStream is = conn.getErrorStream()) {
                    if (is != null) {
                        //res.setErr(objectMapper.readValue(is, JSONError.class));
                    }
                }
            } else {
                try (InputStream is = conn.getInputStream()) {
                    res.setVal(objectMapper.readValue(is, valueType));
                }
            }
            return res;
        } finally {
            conn.disconnect();
        }
    }

    @Test
    public void test0001_getIdentities() throws Exception {
        HTTPResponse<String[]> response = request("GET", "/api/v1/kat/identities", null, String[].class);
        assertEquals(200, response.getStatus());
        assertEquals("O=PartyA, L=London, C=GB", response.getVal()[0]);
    }

    @Test
    public void test0002_createAssetInstances() throws Exception {
        CreateAssetInstanceRequest body = new CreateAssetInstanceRequest();
        String assetDefinitionId = UUID.randomUUID().toString();
        String assetInstanceId = UUID.randomUUID().toString();
        body.setAssetDefinitionID(assetDefinitionId);
        body.setAssetInstanceID(assetInstanceId);
        body.setContentHash("0xaaaabbbbaaaabbbbaaaabbbbaaaabbbb");
        body.setObservers(List.of("O=PartyB,L=New York,C=US"));
        HTTPResponse<AssetResponse> response = request("POST", "/api/v1/kat/createAssetInstance", body, AssetResponse.class);
        assertEquals(200, response.getStatus());
    }

*/
}
