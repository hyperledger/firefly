// Copyright © 2021 Kaleido, Inc.
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

package sqlcommon

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hyperledger-labs/firefly/pkg/database"
	"github.com/hyperledger-labs/firefly/pkg/fftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSubscriptionsE2EWithDB(t *testing.T) {

	s := newQLTestProvider(t)
	defer s.Close()
	ctx := context.Background()

	// Create a new subscription entry
	subscription := &fftypes.Subscription{
		SubscriptionRef: fftypes.SubscriptionRef{
			ID:        nil, // generated for us
			Namespace: "ns1",
			Name:      "subscription1",
		},
		Created: fftypes.Now(),
	}

	s.callbacks.On("SubscriptionCreated", mock.Anything).Return()
	s.callbacks.On("SubscriptionDeleted", mock.Anything).Return()

	err := s.UpsertSubscription(ctx, subscription, true)
	assert.NoError(t, err)

	// Check we get the exact same subscription back
	subscriptionRead, err := s.GetSubscriptionByName(ctx, subscription.Namespace, subscription.Name)
	assert.NoError(t, err)
	assert.NotNil(t, subscriptionRead)
	subscriptionJson, _ := json.Marshal(&subscription)
	subscriptionReadJson, _ := json.Marshal(&subscriptionRead)
	assert.Equal(t, string(subscriptionJson), string(subscriptionReadJson))

	// Update the subscription (this is testing what's possible at the database layer,
	// and does not account for the verification that happens at the higher level)
	newest := fftypes.SubOptsFirstEventNewest
	fifty := uint16(50)
	subOpts := fftypes.SubscriptionOptions{
		SubscriptionCoreOptions: fftypes.SubscriptionCoreOptions{
			FirstEvent: &newest,
			ReadAhead:  &fifty,
		},
	}
	subOpts.TransportOptions()["my-transport-option"] = true
	subscriptionUpdated := &fftypes.Subscription{
		SubscriptionRef: fftypes.SubscriptionRef{
			ID:        fftypes.NewUUID(), // will fail with us trying to update this
			Namespace: "ns1",
			Name:      "subscription1",
		},
		Transport: "websockets",
		Filter: fftypes.SubscriptionFilter{
			Events: string(fftypes.EventTypeMessageConfirmed),
			Topics: "topics.*",
			Tag:    "tag.*",
			Group:  "group.*",
		},
		Options: subOpts,
		Created: fftypes.Now(),
	}

	// Rejects attempt to update ID
	err = s.UpsertSubscription(context.Background(), subscriptionUpdated, true)
	assert.Equal(t, database.IDMismatch, err)

	// Blank out the ID and retry
	subscriptionUpdated.ID = nil
	err = s.UpsertSubscription(context.Background(), subscriptionUpdated, true)
	assert.NoError(t, err)

	// Check we get the exact same data back - note the removal of one of the subscription elements
	subscriptionRead, err = s.GetSubscriptionByID(ctx, subscription.ID)
	assert.NoError(t, err)
	subscriptionJson, _ = json.Marshal(&subscriptionUpdated)
	subscriptionReadJson, _ = json.Marshal(&subscriptionRead)
	assert.Equal(t, string(subscriptionJson), string(subscriptionReadJson))
	assert.Equal(t, true, subscriptionRead.Options.TransportOptions()["my-transport-option"])

	// Query back the subscription
	fb := database.SubscriptionQueryFactory.NewFilter(ctx)
	filter := fb.And(
		fb.Eq("namespace", subscriptionUpdated.Namespace),
		fb.Eq("name", subscriptionUpdated.Name),
	)
	subscriptionRes, err := s.GetSubscriptions(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(subscriptionRes))
	subscriptionReadJson, _ = json.Marshal(subscriptionRes[0])
	assert.Equal(t, string(subscriptionJson), string(subscriptionReadJson))

	// Update
	updateTime := fftypes.Now()
	up := database.SubscriptionQueryFactory.NewUpdate(ctx).Set("created", updateTime)
	err = s.UpdateSubscription(ctx, subscriptionUpdated.Namespace, subscriptionUpdated.Name, up)
	assert.NoError(t, err)

	// Test find updated value
	filter = fb.And(
		fb.Eq("name", subscriptionUpdated.Name),
		fb.Eq("created", updateTime.String()),
	)
	subscriptions, err := s.GetSubscriptions(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(subscriptions))

	// Test delete, and refind no return
	err = s.DeleteSubscriptionByID(ctx, subscriptionUpdated.ID)
	assert.NoError(t, err)
	subscriptions, err = s.GetSubscriptions(ctx, filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(subscriptions))

	s.callbacks.AssertExpectations(t)
}

func TestUpsertSubscriptionFailBegin(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertSubscription(context.Background(), &fftypes.Subscription{}, true)
	assert.Regexp(t, "FF10114", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertSubscriptionFailSelect(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertSubscription(context.Background(), &fftypes.Subscription{SubscriptionRef: fftypes.SubscriptionRef{Name: "name1"}}, true)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertSubscriptionFailInsert(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{}))
	mock.ExpectExec("INSERT .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertSubscription(context.Background(), &fftypes.Subscription{SubscriptionRef: fftypes.SubscriptionRef{Name: "name1"}}, true)
	assert.Regexp(t, "FF10116", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertSubscriptionFailUpdate(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"name"}).
		AddRow("name1"))
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	err := s.UpsertSubscription(context.Background(), &fftypes.Subscription{SubscriptionRef: fftypes.SubscriptionRef{Name: "name1"}}, true)
	assert.Regexp(t, "FF10117", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpsertSubscriptionFailCommit(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"name"}))
	mock.ExpectExec("INSERT .*").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("pop"))
	err := s.UpsertSubscription(context.Background(), &fftypes.Subscription{SubscriptionRef: fftypes.SubscriptionRef{Name: "name1"}}, true)
	assert.Regexp(t, "FF10119", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSubscriptionByIDSelectFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	_, err := s.GetSubscriptionByName(context.Background(), "ns1", "name1")
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSubscriptionByIDNotFound(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"namespace", "name"}))
	msg, err := s.GetSubscriptionByName(context.Background(), "ns1", "name1")
	assert.NoError(t, err)
	assert.Nil(t, msg)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSubscriptionByIDScanFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"namespace"}).AddRow("only one"))
	_, err := s.GetSubscriptionByName(context.Background(), "ns1", "name1")
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSubscriptionQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnError(fmt.Errorf("pop"))
	f := database.SubscriptionQueryFactory.NewFilter(context.Background()).Eq("name", "")
	_, err := s.GetSubscriptions(context.Background(), f)
	assert.Regexp(t, "FF10115", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSubscriptionBuildQueryFail(t *testing.T) {
	s, _ := newMockProvider().init()
	f := database.SubscriptionQueryFactory.NewFilter(context.Background()).Eq("name", map[bool]bool{true: false})
	_, err := s.GetSubscriptions(context.Background(), f)
	assert.Regexp(t, "FF10149.*type", err)
}

func TestGetSubscriptionReadMessageFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectQuery("SELECT .*").WillReturnRows(sqlmock.NewRows([]string{"ntype"}).AddRow("only one"))
	f := database.SubscriptionQueryFactory.NewFilter(context.Background()).Eq("name", "")
	_, err := s.GetSubscriptions(context.Background(), f)
	assert.Regexp(t, "FF10121", err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscriptionUpdateBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	u := database.SubscriptionQueryFactory.NewUpdate(context.Background()).Set("name", "anything")
	err := s.UpdateSubscription(context.Background(), "ns1", "name1", u)
	assert.Regexp(t, "FF10114", err)
}

func TestSubscriptionUpdateBuildQueryFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	u := database.SubscriptionQueryFactory.NewUpdate(context.Background()).Set("name", map[bool]bool{true: false})
	err := s.UpdateSubscription(context.Background(), "ns1", "name1", u)
	assert.Regexp(t, "FF10149.*name", err)
}

func TestSubscriptionUpdateFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*").WillReturnError(fmt.Errorf("pop"))
	mock.ExpectRollback()
	u := database.SubscriptionQueryFactory.NewUpdate(context.Background()).Set("name", fftypes.NewUUID())
	err := s.UpdateSubscription(context.Background(), "ns1", "name1", u)
	assert.Regexp(t, "FF10117", err)
}

func TestSubscriptionDeleteBeginFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin().WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteSubscriptionByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10114", err)
}

func TestSubscriptionDeleteFail(t *testing.T) {
	s, mock := newMockProvider().init()
	mock.ExpectBegin()
	mock.ExpectExec("DELETE .*").WillReturnError(fmt.Errorf("pop"))
	err := s.DeleteSubscriptionByID(context.Background(), fftypes.NewUUID())
	assert.Regexp(t, "FF10118", err)
}
