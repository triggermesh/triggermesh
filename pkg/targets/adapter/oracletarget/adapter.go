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

package oracletarget

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/functions"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/metrics"
)

// NewTarget constructs a target's adapter.
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.OracleTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	fn := env.OracleFunction
	provider := common.NewRawConfigurationProvider(env.TenantOCID, env.UserOCID, env.OracleRegion,
		env.OracleAPIKeyFingerprint, env.OracleAPIKey, &env.OracleAPIKeyPassphrase)

	return &oracleAdapter{
		provider: provider,
		fn:       fn,
		ceClient: ceClient,
		logger:   logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*oracleAdapter)(nil)

type oracleAdapter struct {
	provider     common.ConfigurationProvider
	fnClient     functions.FunctionsInvokeClient
	mgmtClient   functions.FunctionsManagementClient
	funcMetadata functions.GetFunctionResponse
	fn           string

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

func (a *oracleAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Oracle adapter")

	// Need to obtain the function endpoint, and then create the client
	mgmtClient, err := functions.NewFunctionsManagementClientWithConfigurationProvider(a.provider)
	if err != nil {
		a.logger.Errorw("Failed to initialize oracle functions mgmt client", zap.Error(err))
		return err
	}
	a.mgmtClient = mgmtClient

	funcRequest := functions.GetFunctionRequest{
		FunctionId: &a.fn,
	}

	funcResponse, err := mgmtClient.GetFunction(ctx, funcRequest)
	if err != nil {
		a.logger.Errorw("Failed to obtain functions metadata ", zap.Error(err))
		return err
	}
	a.funcMetadata = funcResponse

	client, err := functions.NewFunctionsInvokeClientWithConfigurationProvider(a.provider, *a.funcMetadata.InvokeEndpoint)
	if err != nil {
		return err
	}
	a.fnClient = client

	if err := a.ceClient.StartReceiver(ctx, a.dispatch); err != nil {
		return err
	}
	return nil
}

func (a *oracleAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	requestPayload := io.NopCloser(bytes.NewReader(event.Data()))

	request := functions.InvokeFunctionRequest{
		FunctionId:         &a.fn,
		InvokeFunctionBody: requestPayload,
	}

	response, err := a.fnClient.InvokeFunction(ctx, request)
	if err != nil {
		a.logger.Errorw("Error invoking function", zap.Error(err))
		return nil, cloudevents.ResultNACK
	}

	if response.HTTPResponse().StatusCode != http.StatusOK {
		a.logger.Errorf("Invalid response status for function invocation: %d", response.HTTPResponse().Status)
		return nil, cloudevents.ResultNACK
	}

	defer a.responseCleanup(response)

	respBody, err := io.ReadAll(response.Content)
	if err != nil {
		a.logger.Errorw("Error extracting response from function", zap.Error(err))
		return nil, cloudevents.ResultNACK
	}

	a.logger.Debugf("## body: %v\n", string(respBody))
	responseEvent := cloudevents.NewEvent(cloudevents.VersionV1)
	err = responseEvent.SetData("application/json", respBody)
	if err != nil {
		a.logger.Errorw("Error generating response event", zap.Error(err))
		return nil, cloudevents.ResultNACK
	}
	responseEvent.SetType("functions.oracletargets.targets.triggermesh.io")
	responseEvent.SetSource(*a.funcMetadata.ApplicationId)
	responseEvent.SetSubject(*a.funcMetadata.Id)

	return &responseEvent, cloudevents.ResultACK
}

func (a *oracleAdapter) responseCleanup(response functions.InvokeFunctionResponse) {
	err := response.Content.Close()
	if err != nil {
		a.logger.Errorw("Error during response cleanup", zap.Error(err))
	}
}
