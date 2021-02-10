package io.kaleido.kat.server.data;

public class CreateAssetDefinitionRequest extends AssetRequest {
    public String getAssetDefinitionHash() {
        return assetDefinitionHash;
    }

    public void setAssetDefinitionHash(String assetDefinitionHash) {
        this.assetDefinitionHash = assetDefinitionHash;
    }

    public CreateAssetDefinitionRequest() {
    }

    private String assetDefinitionHash;
}
