package io.kaleido.kat.server.data;

public class RegisterMemberRequest extends AssetRequest {
    private String name;
    private String assetTrailInstanceID;
    private String app2appDestination;
    private String docExchangeDestination;

    public RegisterMemberRequest() {
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getAssetTrailInstanceID() {
        return assetTrailInstanceID;
    }

    public void setAssetTrailInstanceID(String assetTrailInstanceID) {
        this.assetTrailInstanceID = assetTrailInstanceID;
    }

    public String getApp2appDestination() {
        return app2appDestination;
    }

    public void setApp2appDestination(String app2appDestination) {
        this.app2appDestination = app2appDestination;
    }

    public String getDocExchangeDestination() {
        return docExchangeDestination;
    }

    public void setDocExchangeDestination(String docExchangeDestination) {
        this.docExchangeDestination = docExchangeDestination;
    }
}
