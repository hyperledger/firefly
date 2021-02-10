package io.kaleido.kat.server.data;

public class CreateStructuredAssetDefinitionRequest extends CreateUnstructuredAssetDefinitionRequest {
    public String getContentSchemaHash() {
        return contentSchemaHash;
    }

    public void setContentSchemaHash(String contentSchemaHash) {
        this.contentSchemaHash = contentSchemaHash;
    }

    public CreateStructuredAssetDefinitionRequest() {
    }

    private String contentSchemaHash;
}
