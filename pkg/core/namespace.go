// Copyright © 2022 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"context"
	"database/sql/driver"
	"encoding/json"

	"github.com/hyperledger/firefly-common/pkg/fftypes"
	"github.com/hyperledger/firefly-common/pkg/i18n"
)

// NamespaceType describes when the namespace was created from local configuration, or broadcast through the network
type NamespaceType = fftypes.FFEnum

var (
	// NamespaceTypeLocal is a namespace that only exists because it was defined in the local configuration of the node
	NamespaceTypeLocal = fftypes.FFEnumValue("namespacetype", "local")
	// NamespaceTypeBroadcast is a namespace that was broadcast through the network. Broadcast namespaces can overwrite a local namespace
	NamespaceTypeBroadcast = fftypes.FFEnumValue("namespacetype", "broadcast")
	// NamespaceTypeSystem is a reserved namespace used by FireFly itself
	NamespaceTypeSystem = fftypes.FFEnumValue("namespacetype", "system")
)

// Namespace is a isolate set of named resources, to allow multiple applications to co-exist in the same network, with the same named objects.
// Can be used for use case segregation, or multi-tenancy.
type Namespace struct {
	ID          *fftypes.UUID    `ffstruct:"Namespace" json:"id" ffexcludeinput:"true"`
	Message     *fftypes.UUID    `ffstruct:"Namespace" json:"message,omitempty" ffexcludeinput:"true"`
	Name        string           `ffstruct:"Namespace" json:"name"`
	Description string           `ffstruct:"Namespace" json:"description"`
	Type        NamespaceType    `ffstruct:"Namespace" json:"type" ffenum:"namespacetype" ffexcludeinput:"true"`
	Created     *fftypes.FFTime  `ffstruct:"Namespace" json:"created" ffexcludeinput:"true"`
	Contracts   FireFlyContracts `ffstruct:"Namespace" json:"fireflyContract" ffexcludeinput:"true"`
}

type FireFlyContracts struct {
	Active     FireFlyContractInfo   `ffstruct:"FireFlyContracts" json:"active"`
	Terminated []FireFlyContractInfo `ffstruct:"FireFlyContracts" json:"terminated,omitempty"`
}

type FireFlyContractInfo struct {
	Index        int              `ffstruct:"FireFlyContractInfo" json:"index"`
	FinalEvent   string           `ffstruct:"FireFlyContractInfo" json:"finalEvent,omitempty"`
	Location     *fftypes.JSONAny `ffstruct:"FireFlyContractInfo" json:"location,omitempty"`
	FirstEvent   string           `ffstruct:"FireFlyContractInfo" json:"firstEvent,omitempty"`
	Subscription string           `ffstruct:"FireFlyContractInfo" json:"subscription,omitempty"`
}

type NamespaceRef struct {
	LocalName  string
	RemoteName string
}

// NetworkActionType is a type of action to perform
type NetworkActionType = fftypes.FFEnum

var (
	// NetworkActionTerminate request all network members to stop using the current contract and move to the next one configured
	NetworkActionTerminate = fftypes.FFEnumValue("networkactiontype", "terminate")
)

type NetworkAction struct {
	Type NetworkActionType `ffstruct:"NetworkAction" json:"type" ffenum:"networkactiontype"`
}

func (ns *Namespace) Validate(ctx context.Context, existing bool) (err error) {
	if err = fftypes.ValidateFFNameField(ctx, ns.Name, "name"); err != nil {
		return err
	}
	if err = fftypes.ValidateLength(ctx, ns.Description, "description", 4096); err != nil {
		return err
	}
	if existing {
		if ns.ID == nil {
			return i18n.NewError(ctx, i18n.MsgNilID)
		}
	}
	return nil
}

func (ns *Namespace) Topic() string {
	return fftypes.TypeNamespaceNameTopicHash("namespace", ns.Name, "")
}

func (ns *Namespace) SetBroadcastMessage(msgID *fftypes.UUID) {
	ns.Message = msgID
}

// Scan implements sql.Scanner
func (fc *FireFlyContracts) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		if len(src) == 0 {
			return nil
		}
		return json.Unmarshal(src, fc)
	case string:
		return fc.Scan([]byte(src))
	default:
		return i18n.NewError(context.Background(), i18n.MsgTypeRestoreFailed, src, fc)
	}
}

// Value implements sql.Valuer
func (fc FireFlyContracts) Value() (driver.Value, error) {
	return json.Marshal(fc)
}
