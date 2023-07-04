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

package eventbridge

import (
	"fmt"

	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	awscore "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eventbridge"
	"github.com/aws/aws-sdk-go/service/eventbridge/eventbridgeiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/aws"
)

// Client is an alias for the EventBridgeAPI interface.
type Client = eventbridgeiface.EventBridgeAPI

// SQSClient is an alias for the SQSAPI interface.
type SQSClient = sqsiface.SQSAPI

// ClientGetter can obtain EventBridge and SQS clients.
type ClientGetter interface {
	Get(*v1alpha1.AWSEventBridgeSource) (Client, SQSClient, error)
}

// NewClientGetter returns a ClientGetter for the given secrets getter.
func NewClientGetter(sg NamespacedSecretsGetter) *ClientGetterWithSecretGetter {
	return &ClientGetterWithSecretGetter{
		sg: sg,
	}
}

// NamespacedSecretsGetter returns a SecretInterface for the given namespace.
type NamespacedSecretsGetter func(namespace string) coreclientv1.SecretInterface

// ClientGetterWithSecretGetter gets EventBridge clients using static
// credentials retrieved using a Secret getter.
type ClientGetterWithSecretGetter struct {
	sg NamespacedSecretsGetter
}

// ClientGetterWithSecretGetter implements ClientGetter.
var _ ClientGetter = (*ClientGetterWithSecretGetter)(nil)

// Get implements ClientGetter.
func (g *ClientGetterWithSecretGetter) Get(src *v1alpha1.AWSEventBridgeSource) (Client, SQSClient, error) {
	var sess *session.Session
	config := &awscore.Config{}

	switch {
	case src.Spec.Auth.Credentials != nil:
		creds, err := aws.Credentials(g.sg(src.Namespace), src.Spec.Auth.Credentials)
		if err != nil {
			return nil, nil, fmt.Errorf("retrieving AWS security credentials: %w", err)
		}
		sess = session.Must(session.NewSession(awscore.NewConfig().
			WithRegion(src.Spec.ARN.Region).
			WithCredentials(credentials.NewStaticCredentialsFromCreds(*creds)),
		))
		if assumeRole := src.Spec.Auth.Credentials.AssumeIAMRole; assumeRole != nil {
			config.Credentials = stscreds.NewCredentials(sess, assumeRole.String())
		}
	case src.Spec.Auth.EksIAMRole != nil || src.Spec.Auth.IAM != nil:
		sess = session.Must(session.NewSession(awscore.NewConfig().
			WithRegion(src.Spec.ARN.Region),
		))
	default:
		return nil, nil, fmt.Errorf("neither AWS security credentials nor IAM Role were specified")
	}

	return eventbridge.New(sess, config), sqs.New(sess, config), nil
}

// ClientGetterFunc allows the use of ordinary functions as ClientGetter.
type ClientGetterFunc func(*v1alpha1.AWSEventBridgeSource) (Client, SQSClient, error)

// ClientGetterFunc implements ClientGetter.
var _ ClientGetter = (ClientGetterFunc)(nil)

// Get implements ClientGetter.
func (f ClientGetterFunc) Get(src *v1alpha1.AWSEventBridgeSource) (Client, SQSClient, error) {
	return f(src)
}
