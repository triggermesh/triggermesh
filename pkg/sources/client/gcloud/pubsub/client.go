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

package pubsub

import (
	"context"
	"fmt"

	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/option"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/secret"
)

// ClientGetter can obtain Google Cloud API clients.
type ClientGetter interface {
	Get(*v1alpha1.GoogleCloudPubSubSource) (*pubsub.Client, error)
}

// NewClientGetter returns a ClientGetter for the given secrets getter.
func NewClientGetter(sg NamespacedSecretsGetter) *ClientGetterWithSecretGetter {
	return &ClientGetterWithSecretGetter{
		sg: sg,
	}
}

// NamespacedSecretsGetter returns a SecretInterface for the given namespace.
type NamespacedSecretsGetter func(namespace string) coreclientv1.SecretInterface

// ClientGetterWithSecretGetter gets Google Cloud clients using static
// credentials retrieved using a Secret getter.
type ClientGetterWithSecretGetter struct {
	sg NamespacedSecretsGetter
}

// ClientGetterWithSecretGetter implements ClientGetter.
var _ ClientGetter = (*ClientGetterWithSecretGetter)(nil)

// Get implements ClientGetter.
func (g *ClientGetterWithSecretGetter) Get(src *v1alpha1.GoogleCloudPubSubSource) (*pubsub.Client, error) {
	project := src.Spec.Topic.Project

	creds := make([]option.ClientOption, 0)
	if src.Spec.Auth.ServiceAccountKey != nil {
		requestedSecrets, err := secret.NewGetter(g.sg(src.Namespace)).Get(*src.Spec.Auth.ServiceAccountKey)
		if err != nil {
			return nil, fmt.Errorf("retrieving Google Cloud service account key: %w", err)
		}
		saKey := []byte(requestedSecrets[0])
		creds = append(creds, option.WithCredentialsJSON(saKey))
	}

	cli, err := pubsub.NewClient(context.Background(), project, creds...)
	if err != nil {
		return nil, fmt.Errorf("creating Google Cloud Pub/Sub API client: %w", err)
	}

	return cli, nil
}

// ClientGetterFunc allows the use of ordinary functions as ClientGetter.
type ClientGetterFunc func(*v1alpha1.GoogleCloudPubSubSource) (*pubsub.Client, error)

// ClientGetterFunc implements ClientGetter.
var _ ClientGetter = (ClientGetterFunc)(nil)

// Get implements ClientGetter.
func (f ClientGetterFunc) Get(src *v1alpha1.GoogleCloudPubSubSource) (*pubsub.Client, error) {
	return f(src)
}
