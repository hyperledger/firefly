package io.kaleido.kat.server.data;

public class CreateAssetInstanceRequest extends AssetRequest {
    private String assetInstanceID;
    private String assetDefinitionID;
    private String contentHash;

    public CreateAssetInstanceRequest() {
    }

    public String getAssetInstanceID() {
        return assetInstanceID;
    }

    public void setAssetInstanceID(String assetInstanceID) {
        this.assetInstanceID = assetInstanceID;
    }

    public String getAssetDefinitionID() {
        return assetDefinitionID;
    }

    public void setAssetDefinitionID(String assetDefinitionID) {
        this.assetDefinitionID = assetDefinitionID;
    }

    public String getContentHash() {
        return contentHash;
    }

    public void setContentHash(String contentHash) {
        this.contentHash = contentHash;
    }
}
