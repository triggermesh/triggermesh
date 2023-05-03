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

package awscognitouserpoolsource

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/uuid"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider/cognitoidentityprovideriface"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/health"
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

	cgnIdentityClient cognitoidentityprovideriface.CognitoIdentityProviderAPI
	ceClient          cloudevents.Client

	arn        arn.ARN
	userPoolID string
}

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.AWSCognitoUserPoolSourceResource.String(),
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

		cgnIdentityClient: cognitoidentityprovider.New(sess, config),
		ceClient:          ceClient,

		arn:        arn,
		userPoolID: common.MustParseCognitoUserPoolResource(arn.Resource),
	}
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	go health.Start(ctx)

	if err := validatePool(a.cgnIdentityClient, a.userPoolID); err != nil {
		return fmt.Errorf("validating user pool: %w", err)
	}

	health.MarkReady()

	a.logger.Infof("Listening to AWS Cognito User Pool: %s", a.userPoolID)

	ctx = pkgadapter.ContextWithMetricTag(ctx, a.mt)

	var latestTimestamp time.Time

	backoff := common.NewBackoff()

	err := backoff.Run(ctx.Done(), func(ctx context.Context) (bool, error) {
		resetBackoff := false
		users, err := a.listUsers()
		if err != nil {
			a.logger.Errorw("Cognito ListUsers failed", zap.Error(err))
			return resetBackoff, err
		}

		users, latestTimestamp = filterByTimestamp(users, latestTimestamp)

		for _, user := range users {
			// we have new users - reset backoff duration
			resetBackoff = true
			err := a.sendCognitoEvent(ctx, user)
			if err != nil {
				a.logger.Errorw("Failed to send CloudEvent", zap.Error(err))
			}
		}
		return resetBackoff, nil
	})

	return err
}

func (a *adapter) listUsers() ([]*cognitoidentityprovider.UserType, error) {
	input := &cognitoidentityprovider.ListUsersInput{
		UserPoolId: &a.userPoolID,
	}
	output, err := a.cgnIdentityClient.ListUsers(input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return nil, fmt.Errorf("cognito ListUsers response is nil")
	}
	return output.Users, nil
}

func filterByTimestamp(users []*cognitoidentityprovider.UserType, latestTimestamp time.Time) ([]*cognitoidentityprovider.UserType, time.Time) {
	var newUsers []*cognitoidentityprovider.UserType
	newLatestTimestamp := latestTimestamp
	for _, user := range users {
		// Get latest modification timestamp from users list
		// and store it in temporary variable
		if user.UserLastModifiedDate.After(newLatestTimestamp) {
			newLatestTimestamp = *user.UserLastModifiedDate
		}
		// latest.isZero() true in first iteration - do not send already existing users.
		// Also, do not send user object if it was not modified after our latest timestamp mark.
		// (we use "not after" because "before" will be always false for the last created user)
		if latestTimestamp.IsZero() || !user.UserLastModifiedDate.After(latestTimestamp) {
			continue
		}
		newUsers = append(newUsers, user)
	}
	return newUsers, newLatestTimestamp
}

func (a *adapter) sendCognitoEvent(ctx context.Context, user *cognitoidentityprovider.UserType) error {
	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetSubject(a.userPoolID)
	event.SetSource(a.arn.String())
	event.SetID(string(uuid.NewUUID()))
	event.SetType(v1alpha1.AWSEventType(a.arn.Service, v1alpha1.AWSCognitoUserPoolGenericEventType))
	if err := event.SetData(cloudevents.ApplicationJSON, user); err != nil {
		return fmt.Errorf("failed to set event data: %w", err)
	}

	if result := a.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
		return result
	}
	return nil
}

// validatePool ensures the pool with the given ID exists.
func validatePool(cli cognitoidentityprovideriface.CognitoIdentityProviderAPI, poolID string) error {
	_, err := cli.DescribeUserPool(&cognitoidentityprovider.DescribeUserPoolInput{
		UserPoolId: &poolID,
	})
	return err
}
