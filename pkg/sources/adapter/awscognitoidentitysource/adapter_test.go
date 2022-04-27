/*
Copyright 2022 TriggerMesh Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package awscognitoidentitysource

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cognitoidentity"
	"github.com/aws/aws-sdk-go/service/cognitoidentity/cognitoidentityiface"
	"github.com/aws/aws-sdk-go/service/cognitosync"
	"github.com/aws/aws-sdk-go/service/cognitosync/cognitosynciface"

	adaptertest "knative.dev/eventing/pkg/adapter/v2/test"
	loggingtesting "knative.dev/pkg/logging/testing"
)

type mockedCognitoIdentityClient struct {
	cognitoidentityiface.CognitoIdentityAPI
	listIdentitiesOutput      cognitoidentity.ListIdentitiesOutput
	listIdentitiesOutputError error
}

func (m mockedCognitoIdentityClient) ListIdentities(in *cognitoidentity.ListIdentitiesInput) (*cognitoidentity.ListIdentitiesOutput, error) {
	return &m.listIdentitiesOutput, m.listIdentitiesOutputError
}

type mockedCognitoSyncClient struct {
	cognitosynciface.CognitoSyncAPI
	listDatasetsOutput      cognitosync.ListDatasetsOutput
	listRecordsOutput       cognitosync.ListRecordsOutput
	listDatasetsOutputError error
	listRecordsOutputError  error
}

func (m mockedCognitoSyncClient) ListDatasets(in *cognitosync.ListDatasetsInput) (*cognitosync.ListDatasetsOutput, error) {
	return &m.listDatasetsOutput, m.listDatasetsOutputError
}

func (m mockedCognitoSyncClient) ListRecords(in *cognitosync.ListRecordsInput) (*cognitosync.ListRecordsOutput, error) {
	return &m.listRecordsOutput, m.listRecordsOutputError
}

func TestGetIdentities(t *testing.T) {
	a := &adapter{
		logger: loggingtesting.TestLogger(t),
	}

	a.cgnIdentityClient = mockedCognitoIdentityClient{
		listIdentitiesOutput:      cognitoidentity.ListIdentitiesOutput{},
		listIdentitiesOutputError: errors.New("fake ListIdentities error"),
	}

	identities, err := a.getIdentities()
	assert.Error(t, err)
	assert.Equal(t, 0, len(identities))

	a.cgnIdentityClient = mockedCognitoIdentityClient{
		listIdentitiesOutput: cognitoidentity.ListIdentitiesOutput{
			Identities: []*cognitoidentity.IdentityDescription{{}, {}},
		},
		listIdentitiesOutputError: nil,
	}

	identities, err = a.getIdentities()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(identities))
}

func TestGetDatasets(t *testing.T) {
	a := &adapter{
		logger: loggingtesting.TestLogger(t),
	}

	identities := []*cognitoidentity.IdentityDescription{{
		IdentityId: aws.String("1"),
	}}

	a.cgnSyncClient = mockedCognitoSyncClient{
		listDatasetsOutput:      cognitosync.ListDatasetsOutput{},
		listDatasetsOutputError: errors.New("fake ListDatasets error"),
	}

	datasets, err := a.getDatasets(identities)
	assert.Error(t, err)
	assert.Equal(t, 0, len(datasets))

	a.cgnSyncClient = mockedCognitoSyncClient{
		listDatasetsOutput: cognitosync.ListDatasetsOutput{
			Datasets: []*cognitosync.Dataset{{}, {}},
		},
		listDatasetsOutputError: nil,
	}

	datasets, err = a.getDatasets(identities)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(datasets))
}

func TestGetRecords(t *testing.T) {
	a := &adapter{
		logger: loggingtesting.TestLogger(t),
	}

	dataset := cognitosync.Dataset{}

	a.cgnSyncClient = mockedCognitoSyncClient{
		listRecordsOutput:      cognitosync.ListRecordsOutput{},
		listRecordsOutputError: errors.New("fake ListRecords error"),
	}

	records, err := a.getRecords(&dataset)
	assert.Error(t, err)
	assert.Equal(t, 0, len(records))

	a.cgnSyncClient = mockedCognitoSyncClient{
		listRecordsOutput: cognitosync.ListRecordsOutput{
			Records: []*cognitosync.Record{{}, {}},
		},
		listRecordsOutputError: nil,
	}

	records, err = a.getRecords(&dataset)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(records))
}

func TestSendCognitoEvent(t *testing.T) {
	ceClient := adaptertest.NewTestClient()

	a := &adapter{
		logger:         loggingtesting.TestLogger(t),
		identityPoolID: "fooPool",
		ceClient:       ceClient,
	}

	dataset := cognitosync.Dataset{
		DatasetName: aws.String("foo"),
		IdentityId:  aws.String("3234234"),
	}
	records := []*cognitosync.Record{}

	ctx := context.Background()

	err := a.sendCognitoEvent(ctx, &dataset, records)
	assert.NoError(t, err)

	gotEvents := ceClient.Sent()
	assert.Len(t, gotEvents, 1, "Expected 1 event, got %d", len(gotEvents))

	wantData := `{"CreationDate":null,"DataStorage":null,"DatasetName":"foo","IdentityID":"3234234","LastModifiedBy":null,"LastModifiedDate":null,"NumRecords":null,"EventType":"SyncTrigger","Region":"","IdentityPoolID":"fooPool","DatasetRecords":[]}`
	gotData := string(gotEvents[0].Data())
	assert.EqualValues(t, wantData, gotData, "Expected event %q, got %q", wantData, gotData)
}
