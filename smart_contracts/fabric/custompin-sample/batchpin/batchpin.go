package batchpin

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type Args struct {
	UUIDs      string   `json:"uuids"`
	BatchHash  string   `json:"batchHash"`
	PayloadRef string   `json:"payloadRef"`
	Contexts   []string `json:"contexts"`
}

type Event struct {
	Signer     string               `json:"signer"`
	Timestamp  *timestamp.Timestamp `json:"timestamp"`
	Action     string               `json:"action"`
	UUIDs      string               `json:"uuids"`
	BatchHash  string               `json:"batchHash"`
	PayloadRef string               `json:"payloadRef"`
	Contexts   []string             `json:"contexts"`
}

func BuildEvent(ctx contractapi.TransactionContextInterface, args *Args) (*Event, error) {
	cid := ctx.GetClientIdentity()
	id, err := cid.GetID()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain client identity's ID: %s", err)
	}
	idString, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		return nil, fmt.Errorf("failed to decode client identity ID: %s", err)
	}
	mspID, err := cid.GetMSPID()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain client identity's MSP ID: %s", err)
	}
	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction timestamp: %s", err)
	}
	return &Event{
		Signer:     fmt.Sprintf("%s::%s", mspID, idString),
		Timestamp:  timestamp,
		UUIDs:      args.UUIDs,
		BatchHash:  args.BatchHash,
		PayloadRef: args.PayloadRef,
		Contexts:   args.Contexts,
	}, nil
}

func BuildEventFromString(ctx contractapi.TransactionContextInterface, data string) (*Event, error) {
	var args Args
	if err := json.Unmarshal([]byte(data), &args); err != nil {
		return nil, err
	}
	return BuildEvent(ctx, &args)
}
