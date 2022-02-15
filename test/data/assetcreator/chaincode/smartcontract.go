package chaincode

import (
	"encoding/json"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

type Asset struct {
	Name string `json:"name"`
}

func (s *SmartContract) CreateAsset(ctx contractapi.TransactionContextInterface, name string) error {
	asset := Asset{Name: name}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	ctx.GetStub().SetEvent("AssetCreated", assetJSON)
	return ctx.GetStub().PutState(name, assetJSON)
}
