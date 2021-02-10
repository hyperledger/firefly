package io.kaleido.kat.server.data;

public class AssetResponse {
    public String getTxHash() {
        return txHash;
    }

    public void setTxHash(String txHash) {
        this.txHash = txHash;
    }

    private String txHash;

    public AssetResponse() {
    }
}
