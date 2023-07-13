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

package googlecloudworkflowstarget

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	workflows "cloud.google.com/go/workflows/apiv1beta"
	executions "cloud.google.com/go/workflows/executions/apiv1beta"
	"cloud.google.com/go/workflows/executions/apiv1beta/executionspb"
	"google.golang.org/api/option"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.GoogleCloudWorkflowsTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	opts := make([]option.ClientOption, 0)
	if env.ServiceAccountKey != nil {
		opts = append(opts, option.WithCredentialsJSON(env.ServiceAccountKey))
	}

	client, err := workflows.NewClient(ctx, opts...)
	if err != nil {
		logger.Panicf("Failed to create client: %v", err)
	}

	eClient, err := executions.NewClient(ctx, opts...)
	if err != nil {
		logger.Panicf("Failed to create client: %v", err)
	}

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeGoogleCloudWorkflowsRunResponse),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	return &googlecloudworkflowsAdapter{
		client:  client,
		eClient: eClient,

		replier:  replier,
		ceClient: ceClient,
		logger:   logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*googlecloudworkflowsAdapter)(nil)

type googlecloudworkflowsAdapter struct {
	client  *workflows.Client
	eClient *executions.Client

	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

// Returns if stopCh is closed or Send() returns an error.
func (a *googlecloudworkflowsAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Google Cloud Workflows Adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *googlecloudworkflowsAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	switch typ := event.Type(); typ {
	case v1alpha1.EventTypeGoogleCloudWorkflowsRun:
		return a.runWorkflow(ctx, event)
	default:
		return a.replier.Error(&event, targetce.ErrorCodeEventContext, fmt.Errorf("event type %q is not supported", typ), nil)
	}
}

func (a *googlecloudworkflowsAdapter) runWorkflow(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	rjp := &RunJobEvent{}
	if err := event.DataAs(rjp); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}

	req := &executionspb.CreateExecutionRequest{
		Parent: rjp.Parent,
		Execution: &executionspb.Execution{
			Name:     rjp.ExecutionName,
			Argument: rjp.Argument,
		},
	}
	resp, err := a.eClient.CreateExecution(ctx, req)
	if err != nil {
		return nil, err
	}

	return a.replier.Ok(&event, resp)
}
