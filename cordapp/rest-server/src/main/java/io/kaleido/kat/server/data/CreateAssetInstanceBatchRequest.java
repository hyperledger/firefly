package io.kaleido.kat.server.data;

public class CreateAssetInstanceBatchRequest extends AssetRequest {
    private String batchHash;

    public CreateAssetInstanceBatchRequest() {
    }

    public String getBatchHash() {
        return batchHash;
    }

    public void setBatchHash(String batchHash) {
        this.batchHash = batchHash;
    }
}
