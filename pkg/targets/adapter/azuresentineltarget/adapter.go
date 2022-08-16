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

package azuresentineltarget

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

type adapter struct {
	client         *http.Client
	clientID       string
	tenantID       string
	azureCreds     string
	subscriptionID string
	resourceGroup  string
	workspace      string
	clientSecret   string

	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
	mt       *pkgadapter.MetricTag
	sr       *metrics.EventProcessingStatsReporter
}

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: "azuresentineltargets",
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeAzureSentinelTargetGenericResponse),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	return &adapter{
		client:         http.DefaultClient,
		clientID:       env.ClientID,
		tenantID:       env.TenantID,
		azureCreds:     env.ClientSecret,
		subscriptionID: env.SubscriptionID,
		resourceGroup:  env.ResourceGroup,
		workspace:      env.Workspace,
		clientSecret:   env.ClientSecret,

		replier:  replier,
		ceClient: ceClient,
		logger:   logger,
		mt:       mt,
		sr:       metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*adapter)(nil)

func (a *adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting AzureSentinel Target Adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *adapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	ceTypeTag := metrics.TagEventType(event.Type())
	ceSrcTag := metrics.TagEventSource(event.Source())

	start := time.Now()
	defer func() {
		a.sr.ReportProcessingLatency(time.Since(start), ceTypeTag, ceSrcTag)
	}()

	typ := event.Type()
	if typ != v1alpha1.EventTypeAzureSentinelTargetIncident {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeEventContext, fmt.Errorf("event type %q is not supported", typ), nil)
	}

	i := &Incident{}
	if err := event.DataAs(i); err != nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		a.logger.Errorw("decoding event: %v", zap.Error(err))
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		a.logger.Errorw("creating Azure authorizer: %v", zap.Error(err))
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	reqBody, err := json.Marshal(*i)
	if err != nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "marshaling request for retrieving an access token")
	}

	rURL := fmt.Sprintf("https://management.azure.com/"+
		"subscriptions/%s/resourceGroups/%s/"+
		"providers/Microsoft.OperationalInsights/workspaces/%s/"+
		"providers/Microsoft.SecurityInsights/incidents/%s?api-version=2020-01-01",
		a.subscriptionID, a.resourceGroup, a.workspace, uuid.New().String())
	request, err := http.NewRequest(http.MethodPut, rURL, bytes.NewBuffer(reqBody))
	if err != nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "creating request token")
	}

	request.Header.Set("Content-Type", "application/json")
	req, err := autorest.Prepare(request,
		authorizer.WithAuthorization())
	if err != nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "preparing request")
	}

	res, err := autorest.Send(req)
	if err != nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "sending request")
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "reading response body")
	}

	if res.StatusCode != 201 {
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "invalid response from Azure: "+string(body))
	}

	a.sr.ReportProcessingSuccess(ceTypeTag, ceSrcTag)
	return a.replier.Ok(&event, body)
}
