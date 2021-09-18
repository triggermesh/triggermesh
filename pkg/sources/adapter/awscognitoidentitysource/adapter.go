/*
Copyright 2019-2020 TriggerMesh Inc.

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
	"fmt"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentity"
	"github.com/aws/aws-sdk-go/service/cognitoidentity/cognitoidentityiface"
	"github.com/aws/aws-sdk-go/service/cognitosync"
	"github.com/aws/aws-sdk-go/service/cognitosync/cognitosynciface"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common"
)

// envConfig is a set parameters sourced from the environment for the source's
// adapter.
type envConfig struct {
	pkgadapter.EnvConfig

	ARN string `envconfig:"ARN" required:"true"`
}

// adapter implements the source's adapter.
type adapter struct {
	logger *zap.SugaredLogger

	cgnIdentityClient cognitoidentityiface.CognitoIdentityAPI
	cgnSyncClient     cognitosynciface.CognitoSyncAPI
	ceClient          cloudevents.Client

	arn            arn.ARN
	identityPoolID string
}

// NewEnvConfig returns an accessor for the source's adapter envConfig.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter returns a constructor for the source's adapter.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	env := envAcc.(*envConfig)

	arn := common.MustParseARN(env.ARN)

	cfg := session.Must(session.NewSession(aws.NewConfig().
		WithRegion(arn.Region).
		WithMaxRetries(5),
	))

	return &adapter{
		logger: logger,

		cgnIdentityClient: cognitoidentity.New(cfg),
		cgnSyncClient:     cognitosync.New(cfg),
		ceClient:          ceClient,

		arn:            arn,
		identityPoolID: common.MustParseCognitoIdentityResource(arn.Resource),
	}
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	a.logger.Infof("Listening to AWS Cognito stream for Identity: %s", a.identityPoolID)

	backoff := common.NewBackoff()

	err := backoff.Run(ctx.Done(), func(ctx context.Context) (bool, error) {
		resetBackoff := false
		identities, err := a.getIdentities()
		if err != nil {
			a.logger.Error(err)
		}

		datasets, err := a.getDatasets(identities)
		if err != nil {
			a.logger.Error(err)
		}

		for _, dataset := range datasets {
			resetBackoff = true
			records, err := a.getRecords(dataset)
			if err != nil {
				a.logger.Error(err)
				continue
			}

			err = a.sendCognitoEvent(dataset, records)
			if err != nil {
				a.logger.Errorf("SendCloudEvent failed: %v", err)
			}
		}
		return resetBackoff, nil
	})

	return err
}

func (a *adapter) getIdentities() ([]*cognitoidentity.IdentityDescription, error) {
	identities := []*cognitoidentity.IdentityDescription{}

	listIdentitiesInput := cognitoidentity.ListIdentitiesInput{
		MaxResults:     aws.Int64(1),
		IdentityPoolId: &a.identityPoolID,
	}

	for {
		listIdentitiesOutput, err := a.cgnIdentityClient.ListIdentities(&listIdentitiesInput)
		if err != nil {
			return identities, err
		}

		identities = append(identities, listIdentitiesOutput.Identities...)

		listIdentitiesInput.NextToken = listIdentitiesOutput.NextToken
		if listIdentitiesOutput.NextToken == nil {
			break
		}
	}

	return identities, nil
}

func (a *adapter) getDatasets(identities []*cognitoidentity.IdentityDescription) ([]*cognitosync.Dataset, error) {
	datasets := []*cognitosync.Dataset{}

	for _, identity := range identities {
		listDatasetsInput := cognitosync.ListDatasetsInput{
			IdentityPoolId: &a.identityPoolID,
			IdentityId:     identity.IdentityId,
		}

		for {
			listDatasetsOutput, err := a.cgnSyncClient.ListDatasets(&listDatasetsInput)
			if err != nil {
				return datasets, err
			}

			datasets = append(datasets, listDatasetsOutput.Datasets...)

			listDatasetsInput.NextToken = listDatasetsOutput.NextToken
			if listDatasetsOutput.NextToken == nil {
				break
			}
		}
	}

	return datasets, nil
}

func (a *adapter) getRecords(dataset *cognitosync.Dataset) ([]*cognitosync.Record, error) {
	records := []*cognitosync.Record{}

	input := cognitosync.ListRecordsInput{
		DatasetName:    dataset.DatasetName,
		IdentityId:     dataset.IdentityId,
		IdentityPoolId: &a.identityPoolID,
	}

	for {
		recordsOutput, err := a.cgnSyncClient.ListRecords(&input)
		if err != nil {
			return records, err
		}

		records = append(records, recordsOutput.Records...)

		input.NextToken = recordsOutput.NextToken
		if recordsOutput.NextToken == nil {
			break
		}
	}

	return records, nil
}

func (a *adapter) sendCognitoEvent(dataset *cognitosync.Dataset, records []*cognitosync.Record) error {
	a.logger.Info("Processing Dataset: ", *dataset.DatasetName)

	data := &CognitoIdentitySyncEvent{
		CreationDate:     dataset.CreationDate,
		DataStorage:      dataset.DataStorage,
		DatasetName:      dataset.DatasetName,
		IdentityID:       dataset.IdentityId,
		LastModifiedBy:   dataset.LastModifiedBy,
		LastModifiedDate: dataset.LastModifiedDate,
		NumRecords:       dataset.NumRecords,
		EventType:        aws.String("SyncTrigger"),
		Region:           &a.arn.Region,
		IdentityPoolID:   &a.identityPoolID,
		DatasetRecords:   records,
	}

	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetType(v1alpha1.AWSEventType(a.arn.Service, v1alpha1.AWSCognitoIdentityGenericEventType))
	event.SetSubject(*dataset.DatasetName)
	event.SetSource(a.arn.String())
	event.SetID(*dataset.IdentityId)
	if err := event.SetData(cloudevents.ApplicationJSON, data); err != nil {
		return fmt.Errorf("failed to set event data: %w", err)
	}

	if result := a.ceClient.Send(context.Background(), event); !cloudevents.IsACK(result) {
		return result
	}
	return nil
}
