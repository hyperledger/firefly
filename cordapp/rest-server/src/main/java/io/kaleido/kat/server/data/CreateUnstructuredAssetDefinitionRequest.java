package io.kaleido.kat.server.data;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.io.Serializable;

public class CreateUnstructuredAssetDefinitionRequest extends AssetRequest implements Serializable {
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

    @JsonProperty(value="isContentPrivate")
    public boolean isContentPrivate() {
        return isContentPrivate;
    }

    public void setContentPrivate(boolean contentPrivate) {
        isContentPrivate = contentPrivate;
    }

    @JsonProperty(value="isContentUnique")
    public boolean isContentUnique() {
        return isContentUnique;
    }

    public void setContentUnique(boolean contentUnique) {
        isContentUnique = contentUnique;
    }
}
