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

package webhooksource

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
)

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.WebhookSourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	env := envAcc.(*envAccessor)

	return &webhookHandler{
		eventType:               env.EventType,
		eventSource:             env.EventSource,
		extensionAttributesFrom: env.EventExtensionAttributesFrom,
		username:                env.BasicAuthUsername,
		password:                env.BasicAuthPassword,
		corsAllowOrigin:         env.CORSAllowOrigin,

		ceClient: ceClient,
		logger:   logging.FromContext(ctx),
		mt:       mt,
	}
}

var _ pkgadapter.Adapter = (*webhookHandler)(nil)
