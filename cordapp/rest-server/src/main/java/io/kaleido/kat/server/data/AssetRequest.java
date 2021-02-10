package io.kaleido.kat.server.data;

import java.util.List;

public class AssetRequest {
    public AssetRequest() {
    }

    public List<String> getParticipants() {
        return participants;
    }

    public void setParticipants(List<String> participants) {
        this.participants = participants;
    }

    private List<String> participants;

}
