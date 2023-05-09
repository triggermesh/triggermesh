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
	"fmt"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentity"
	"github.com/aws/aws-sdk-go/service/cognitoidentity/cognitoidentityiface"
	"github.com/aws/aws-sdk-go/service/cognitosync"
	"github.com/aws/aws-sdk-go/service/cognitosync/cognitosynciface"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common"
)

// envConfig is a set parameters sourced from the environment for the source's
// adapter.
type envConfig struct {
	pkgadapter.EnvConfig

	ARN string `envconfig:"ARN" required:"true"`

	// Assume this IAM Role when access keys provided.
	AssumeIamRole string `envconfig:"AWS_ASSUME_ROLE_ARN"`

	// The environment variables below aren't read from the envConfig struct
	// by the AWS SDK, but rather directly using os.Getenv().
	// They are nevertheless listed here for documentation purposes.
	_ string `envconfig:"AWS_ACCESS_KEY_ID"`
	_ string `envconfig:"AWS_SECRET_ACCESS_KEY"`
}

// adapter implements the source's adapter.
type adapter struct {
	logger *zap.SugaredLogger
	mt     *pkgadapter.MetricTag

	cgnIdentityClient cognitoidentityiface.CognitoIdentityAPI
	cgnSyncClient     cognitosynciface.CognitoSyncAPI
	ceClient          cloudevents.Client

	arn            arn.ARN
	identityPoolID string
}

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.AWSCognitoIdentitySourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	env := envAcc.(*envConfig)

	arn := common.MustParseARN(env.ARN)

	sess := session.Must(session.NewSession(aws.NewConfig().
		WithRegion(arn.Region).
		WithMaxRetries(5),
	))

	config := &aws.Config{}
	if env.AssumeIamRole != "" {
		config.Credentials = stscreds.NewCredentials(sess, env.AssumeIamRole)
	}

	return &adapter{
		logger: logger,
		mt:     mt,

		cgnIdentityClient: cognitoidentity.New(sess, config),
		cgnSyncClient:     cognitosync.New(sess, config),
		ceClient:          ceClient,

		arn:            arn,
		identityPoolID: common.MustParseCognitoIdentityResource(arn.Resource),
	}
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	a.logger.Infof("Listening to AWS Cognito stream for Identity: %s", a.identityPoolID)

	ctx = pkgadapter.ContextWithMetricTag(ctx, a.mt)

	backoff := common.NewBackoff()

	err := backoff.Run(ctx.Done(), func(ctx context.Context) (bool, error) {
		resetBackoff := false
		identities, err := a.getIdentities()
		if err != nil {
			a.logger.Errorw("Unable to get identities", zap.Error(err))
		}

		datasets, err := a.getDatasets(identities)
		if err != nil {
			a.logger.Errorw("Unable to get datasets", zap.Error(err))
		}

		for _, dataset := range datasets {
			resetBackoff = true
			records, err := a.getRecords(dataset)
			if err != nil {
				a.logger.Errorw("Unable to get records", zap.Error(err))
				continue
			}

			err = a.sendCognitoEvent(ctx, dataset, records)
			if err != nil {
				a.logger.Errorw("Failed to send the event", zap.Error(err))
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

func (a *adapter) sendCognitoEvent(ctx context.Context, dataset *cognitosync.Dataset, records []*cognitosync.Record) error {
	a.logger.Debugf("Processing Dataset: %s", *dataset.DatasetName)

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

	if result := a.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
		return result
	}
	return nil
}
