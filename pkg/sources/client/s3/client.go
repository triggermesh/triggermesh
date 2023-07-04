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

package s3

import (
	"fmt"

	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	awscore "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/aws"
)

// Per AWS conventions, a bucket which does not explicitly specify its location
// is located in the "us-east-1" region.
// https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketLocation.html
const defaultS3Region = "us-east-1"

// Client is an alias for the S3API interface.
type Client = s3iface.S3API

// SQSClient is an alias for the SQSAPI interface.
type SQSClient = sqsiface.SQSAPI

// ClientGetter can obtain S3 and SQS clients.
type ClientGetter interface {
	Get(*v1alpha1.AWSS3Source) (Client, SQSClient, error)
}

// NewClientGetter returns a ClientGetter for the given secrets getter.
func NewClientGetter(sg NamespacedSecretsGetter) *ClientGetterWithSecretGetter {
	return &ClientGetterWithSecretGetter{
		sg: sg,
	}
}

// NamespacedSecretsGetter returns a SecretInterface for the given namespace.
type NamespacedSecretsGetter func(namespace string) coreclientv1.SecretInterface

// ClientGetterWithSecretGetter gets S3 clients using static credentials
// retrieved using a Secret getter.
type ClientGetterWithSecretGetter struct {
	sg NamespacedSecretsGetter
}

// ClientGetterWithSecretGetter implements ClientGetter.
var _ ClientGetter = (*ClientGetterWithSecretGetter)(nil)

// Get implements ClientGetter.
func (g *ClientGetterWithSecretGetter) Get(src *v1alpha1.AWSS3Source) (Client, SQSClient, error) {
	var sess *session.Session
	config := &awscore.Config{}

	var creds *credentials.Value
	var err error

	switch {
	case src.Spec.Auth.Credentials != nil:
		if creds, err = aws.Credentials(g.sg(src.Namespace), src.Spec.Auth.Credentials); err != nil {
			return nil, nil, fmt.Errorf("retrieving AWS security credentials: %w", err)
		}
		sess = session.Must(session.NewSession(awscore.NewConfig().
			WithRegion(defaultS3Region).
			WithCredentials(credentials.NewStaticCredentialsFromCreds(*creds)),
		))
		if assumeRole := src.Spec.Auth.Credentials.AssumeIAMRole; assumeRole != nil {
			config.Credentials = stscreds.NewCredentials(sess, assumeRole.String())
		}
	case src.Spec.Auth.EksIAMRole != nil || src.Spec.Auth.IAM != nil:
		sess = session.Must(session.NewSession(awscore.NewConfig().
			WithRegion(defaultS3Region),
		))
	default:
		return nil, nil, fmt.Errorf("neither AWS security credentials nor IAM Role were specified")
	}

	// The ARN of a S3 bucket differs from other ARNs because it doesn't
	// typically include an account ID or region.
	// However, the reconciliation logic *requires* both of these inputs to
	// be able to set an accurate identity-based access policy between the
	// S3 bucket and the reconciled SQS queue (unless the user provides
	// their own SQS queue).
	// To avoid having to handle user-provided credentials in multiple
	// places, we bake the very specific logic of retrieving both the
	// account ID and region into the ClientGetter for the time being.

	region, err := determineS3Region(src, sess, config)
	if err != nil {
		return nil, nil, fmt.Errorf("determining suitable S3 region: %w", err)
	}
	if src.Spec.ARN.Region == "" {
		src.Spec.ARN.Region = region
	}

	if defaultS3Region != region {
		if creds != nil {
			sess = session.Must(session.NewSession(awscore.NewConfig().
				WithCredentials(credentials.NewStaticCredentialsFromCreds(*creds)).
				WithRegion(region),
			))
			if assumeRole := src.Spec.Auth.Credentials.AssumeIAMRole; assumeRole != nil {
				config.Credentials = stscreds.NewCredentials(sess, assumeRole.String())
			}
		} else if src.Spec.Auth.EksIAMRole != nil || src.Spec.Auth.IAM != nil {
			sess = session.Must(session.NewSession(awscore.NewConfig().
				WithRegion(region),
			))
		}
	}

	accID, err := determineBucketOwnerAccount(src, sess, config)
	if err != nil {
		return nil, nil, fmt.Errorf("determining bucket's owner: %w", err)
	}
	if src.Spec.ARN.AccountID == "" {
		src.Spec.ARN.AccountID = accID
	}

	return s3.New(sess, config), sqs.New(sess, config), nil
}

// determineS3Region determines the most suitable region for interacting with
// AWS S3 based on the provided source's spec.
// In order of preference:
// - Value provided in the ARN of the S3 bucket
// - Value provided in the ARN of the SQS queue
// - Value retrieved from the S3 API
func determineS3Region(src *v1alpha1.AWSS3Source, sess *session.Session, config *awscore.Config) (string, error) {
	if src.Spec.ARN.Region != "" {
		return src.Spec.ARN.Region, nil
	}

	if dest := src.Spec.Destination; dest != nil {
		if sqsDest := dest.SQS; sqsDest != nil {
			return sqsDest.QueueARN.Region, nil
		}
	}

	region, err := getBucketRegion(src.Spec.ARN.Resource, sess, config)
	if err != nil {
		return "", fmt.Errorf("getting location of bucket %q: %w", src.Spec.ARN.Resource, err)
	}

	return region, nil
}

// getBucketRegion retrieves the region the provided bucket resides in.
func getBucketRegion(bucketName string, sess *session.Session, config *awscore.Config) (string, error) {
	resp, err := s3.New(sess, config).GetBucketLocation(&s3.GetBucketLocationInput{
		Bucket: &bucketName,
	})
	if err != nil {
		return "", err
	}

	if loc := resp.LocationConstraint; loc != nil {
		return *loc, nil
	}
	return defaultS3Region, nil
}

// determineBucketOwnerAccount determines the ID of the AWS account that owns
// the S3 bucket configured in the given source.
// In order of preference:
// - Value provided in the ARN of the S3 bucket
// - Value provided in the ARN of the SQS queue
// - Value retrieved from the STS API
func determineBucketOwnerAccount(src *v1alpha1.AWSS3Source, sess *session.Session, config *awscore.Config) (string, error) {
	if src.Spec.ARN.AccountID != "" {
		return src.Spec.ARN.AccountID, nil
	}

	if dest := src.Spec.Destination; dest != nil {
		if sqsDest := dest.SQS; sqsDest != nil {
			return sqsDest.QueueARN.AccountID, nil
		}
	}

	accID, err := getCallerAccountID(sess, config)
	if err != nil {
		return "", fmt.Errorf("getting ID of caller: %w", err)
	}

	return accID, nil
}

// getCallerAccountID retrieves the account ID of the caller.
func getCallerAccountID(sess *session.Session, config *awscore.Config) (string, error) {
	resp, err := sts.New(sess, config).GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}

	return *resp.Account, nil
}

// ClientGetterFunc allows the use of ordinary functions as ClientGetter.
type ClientGetterFunc func(*v1alpha1.AWSS3Source) (Client, SQSClient, error)

// ClientGetterFunc implements ClientGetter.
var _ ClientGetter = (ClientGetterFunc)(nil)

// Get implements ClientGetter.
func (f ClientGetterFunc) Get(src *v1alpha1.AWSS3Source) (Client, SQSClient, error) {
	return f(src)
}
