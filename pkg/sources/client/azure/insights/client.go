/*
Copyright 2021 TriggerMesh Inc.

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

package insights

import (
	"fmt"

	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights/insightsapi"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/azure"
)

// EventCategoriesClient is an alias for the EventCategoriesClientAPI interface.
type EventCategoriesClient = insightsapi.EventCategoriesClientAPI

// DiagnosticSettingsClient is an alias for the DiagnosticSettingsClientAPI interface.
type DiagnosticSettingsClient = insightsapi.DiagnosticSettingsClientAPI

// ClientGetter can obtain clients for Azure Insights APIs.
type ClientGetter interface {
	Get(*v1alpha1.AzureActivityLogsSource) (EventCategoriesClient, DiagnosticSettingsClient, error)
}

// NewClientGetter returns a ClientGetter for the given secrets getter.
func NewClientGetter(sg NamespacedSecretsGetter) *ClientGetterWithSecretGetter {
	return &ClientGetterWithSecretGetter{
		sg: sg,
	}
}

type NamespacedSecretsGetter func(namespace string) coreclientv1.SecretInterface

// ClientGetterWithSecretGetter gets Azure clients using static credentials
// retrieved using a Secret getter.
type ClientGetterWithSecretGetter struct {
	sg NamespacedSecretsGetter
}

// ClientGetterWithSecretGetter implements ClientGetter.
var _ ClientGetter = (*ClientGetterWithSecretGetter)(nil)

// Get implements ClientGetter.
func (g *ClientGetterWithSecretGetter) Get(src *v1alpha1.AzureActivityLogsSource) (EventCategoriesClient, DiagnosticSettingsClient, error) {
	authorizer, err := azure.Authorizer(g.sg(src.Namespace), src.Spec.Auth.ServicePrincipal)
	if err != nil {
		return nil, nil, fmt.Errorf("retrieving Azure service principal credentials: %w", err)
	}

	eventCatCli := insights.NewEventCategoriesClient(src.Spec.EventHubID.SubscriptionID)
	eventCatCli.Authorizer = authorizer

	diagSettingsCli := insights.NewDiagnosticSettingsClient(src.Spec.EventHubID.SubscriptionID)
	diagSettingsCli.Authorizer = authorizer

	return eventCatCli, diagSettingsCli, nil
}

// ClientGetterFunc allows the use of ordinary functions as ClientGetter.
type ClientGetterFunc func(*v1alpha1.AzureActivityLogsSource) (EventCategoriesClient, DiagnosticSettingsClient, error)

// ClientGetterFunc implements ClientGetter.
var _ ClientGetter = (ClientGetterFunc)(nil)

// Get implements ClientGetter.
func (f ClientGetterFunc) Get(src *v1alpha1.AzureActivityLogsSource) (EventCategoriesClient, DiagnosticSettingsClient, error) {
	return f(src)
}
