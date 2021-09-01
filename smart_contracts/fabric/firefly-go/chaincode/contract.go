package chaincode

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
	contractapi.Contract
}

type BatchPinEvent struct {
	Signer     string               `json:"signer"`
	Timestamp  *timestamp.Timestamp `json:"timestamp"`
	Namespace  string               `json:"namespace"`
	Uuids      string               `json:"uuids"`
	BatchHash  string               `json:"batchHash"`
	PayloadRef string               `json:"payloadRef"`
	Contexts   []string             `json:"contexts"`
}

func (s *SmartContract) PinBatch(ctx contractapi.TransactionContextInterface, namespace, uuids, batchHash, payloadRef string, contexts []string) error {
	cid := ctx.GetClientIdentity()
	id, err := cid.GetID()
	if err != nil {
		return fmt.Errorf("Failed to obtain client identity's ID: %s", err)
	}
	idString, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		return fmt.Errorf("Failed to decode client identity ID: %s", err)
	}
	mspId, err := cid.GetMSPID()
	if err != nil {
		return fmt.Errorf("Failed to obtain client identity's MSP ID: %s", err)
	}
	timestamp, err := ctx.GetStub().GetTxTimestamp()
	if err != nil {
		return fmt.Errorf("Failed to get transaction timestamp: %s", err)
	}
	event := BatchPinEvent{
		Signer:     fmt.Sprintf("%s::%s", mspId, idString),
		Timestamp:  timestamp,
		Namespace:  namespace,
		Uuids:      uuids,
		BatchHash:  batchHash,
		PayloadRef: payloadRef,
		Contexts:   contexts,
	}
	bytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("Failed to marshal event: %s", err)
	}
	ctx.GetStub().SetEvent("BatchPin", bytes)
	return nil
}
