package io.kaleido.kat.server.data;

public class CreateUnstructuredAssetDefinitionRequest extends AssetRequest {
    private String assetDefinitionID;
    private String name;
    private boolean isContentPrivate;
    private boolean isContentUnique;

    public CreateUnstructuredAssetDefinitionRequest() {
    }

    public String getAssetDefinitionID() {
        return assetDefinitionID;
    }

    public void setAssetDefinitionID(String assetDefinitionID) {
        this.assetDefinitionID = assetDefinitionID;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public boolean isContentPrivate() {
        return isContentPrivate;
    }

    public void setContentPrivate(boolean contentPrivate) {
        isContentPrivate = contentPrivate;
    }

    public boolean isContentUnique() {
        return isContentUnique;
    }

    public void setContentUnique(boolean contentUnique) {
        isContentUnique = contentUnique;
    }
}
