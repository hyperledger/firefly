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

func (s *SmartContract) GetAsset(ctx contractapi.TransactionContextInterface, name string) (string, error) {
	b, err := ctx.GetStub().GetState(name)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
