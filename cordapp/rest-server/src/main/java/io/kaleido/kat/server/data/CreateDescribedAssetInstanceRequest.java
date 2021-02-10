package io.kaleido.kat.server.data;

public class CreateDescribedAssetInstanceRequest extends CreateAssetInstanceRequest {
    private String descriptionHash;

    public CreateDescribedAssetInstanceRequest() {
    }

    public String getDescriptionHash() {
        return descriptionHash;
    }

    public void setDescriptionHash(String descriptionHash) {
        this.descriptionHash = descriptionHash;
    }
}
