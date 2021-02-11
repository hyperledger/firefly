package io.kaleido.kat.server.data;

public class SetAssetInstancePropertyRequest extends AssetRequest {
    private String assetDefinitionID;
    private String assetInstanceID;
    private String key;
    private String value;

    public SetAssetInstancePropertyRequest() {
    }

    public String getAssetInstanceID() {
        return assetInstanceID;
    }

    public void setAssetInstanceID(String assetInstanceID) {
        this.assetInstanceID = assetInstanceID;
    }

    public String getKey() {
        return key;
    }

    public void setKey(String key) {
        this.key = key;
    }

    public String getValue() {
        return value;
    }

    public void setValue(String value) {
        this.value = value;
    }

    public String getAssetDefinitionID() {
        return assetDefinitionID;
    }

    public void setAssetDefinitionID(String assetDefinitionID) {
        this.assetDefinitionID = assetDefinitionID;
    }
}
