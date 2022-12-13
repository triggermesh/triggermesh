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

package transformation

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/triggermesh/triggermesh/pkg/flow/adapter/transformation/common/storage"
	"strings"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/flow"
	"github.com/triggermesh/triggermesh/pkg/apis/flow/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
)

type envConfig struct {
	pkgadapter.EnvConfig

	// Sink URL where to send cloudevents
	Sink string `envconfig:"K_SINK"`

	// Transformation specifications
	TransformationContext string `envconfig:"TRANSFORMATION_CONTEXT"`
	TransformationData    string `envconfig:"TRANSFORMATION_DATA"`
}

// adapter contains Pipelines for CE transformations and CloudEvents client.
type adapter struct {
	ContextPipeline []v1alpha1.Transform
	DataPipeline    []v1alpha1.Transform

	mt *pkgadapter.MetricTag
	sr *metrics.EventProcessingStatsReporter

	sink string

	client cloudevents.Client
	logger *zap.SugaredLogger
}

// ceContext represents CloudEvents context structure but with exported Extensions.
type ceContext struct {
	*cloudevents.EventContextV1 `json:",inline"`
	Extensions                  map[string]interface{} `json:"Extensions,omitempty"`
}

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: flow.TransformationResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envConfig)

	trnContext, trnData := []v1alpha1.Transform{}, []v1alpha1.Transform{}
	err := json.Unmarshal([]byte(env.TransformationContext), &trnContext)
	if err != nil {
		logger.Fatalf("Cannot unmarshal context transformation env variable: %v", err)
	}
	err = json.Unmarshal([]byte(env.TransformationData), &trnData)
	if err != nil {
		logger.Fatalf("Cannot unmarshal data transformation env variable: %v", err)
	}

	return &adapter{
		mt:              mt,
		sr:              metrics.MustNewEventProcessingStatsReporter(mt),
		ContextPipeline: trnContext,
		DataPipeline:    trnData,
		sink:            env.Sink,
		client:          ceClient,
		logger:          logger,
	}
}

// Start runs CloudEvent receiver and applies transformation Pipeline
// on incoming events.
func (t *adapter) Start(ctx context.Context) error {
	t.logger.Info("Starting Transformation adapter")

	var receiver interface{}
	receiver = t.receiveAndReply
	if t.sink != "" {
		ctx = cloudevents.ContextWithTarget(ctx, t.sink)
		receiver = t.receiveAndSend
	}

	ctx = pkgadapter.ContextWithMetricTag(ctx, t.mt)

	return t.client.StartReceiver(ctx, receiver)
}

func (t *adapter) receiveAndReply(event cloudevents.Event) (*cloudevents.Event, error) {
	ceTypeTag := metrics.TagEventType(event.Type())
	ceSrcTag := metrics.TagEventSource(event.Source())

	start := time.Now()
	defer func() {
		t.sr.ReportProcessingLatency(time.Since(start), ceTypeTag, ceSrcTag)
	}()

	result, err := t.applyTransformations(event)
	if err != nil {
		t.sr.ReportProcessingError(false, ceTypeTag, ceSrcTag)
	} else {
		t.sr.ReportProcessingSuccess(ceTypeTag, ceSrcTag)
	}

	return result, err
}

func (t *adapter) receiveAndSend(ctx context.Context, event cloudevents.Event) error {
	ceTypeTag := metrics.TagEventType(event.Type())
	ceSrcTag := metrics.TagEventSource(event.Source())

	start := time.Now()
	defer func() {
		t.sr.ReportProcessingLatency(time.Since(start), ceTypeTag, ceSrcTag)
	}()

	result, err := t.applyTransformations(event)
	if err != nil {
		t.sr.ReportProcessingError(false, ceTypeTag, ceSrcTag)
		return err
	}

	if result := t.client.Send(ctx, *result); !cloudevents.IsACK(result) {
		t.sr.ReportProcessingError(false, ceTypeTag, ceSrcTag)
		return result
	}

	t.sr.ReportProcessingSuccess(ceTypeTag, ceSrcTag)
	return nil
}

func (t *adapter) applyTransformations(event cloudevents.Event) (*cloudevents.Event, error) {
	contextPl, err := newPipeline(t.ContextPipeline)
	if err != nil {
		t.logger.Fatalf("Cannot create context transformation pipeline: %v", err)
	}

	dataPl, err := newPipeline(t.DataPipeline)
	if err != nil {
		t.logger.Fatalf("Cannot create data transformation pipeline: %v", err)
	}

	// set shared storage, scoped too the application function
	sharedStorage := storage.New()

	contextPl.setStorage(sharedStorage)
	dataPl.setStorage(sharedStorage)

	// HTTPTargets sets content type from HTTP headers, i.e.:
	// "datacontenttype: application/json; charset=utf-8"
	// so we must use "contains" instead of strict equality
	if !strings.Contains(event.DataContentType(), cloudevents.ApplicationJSON) {
		err := fmt.Errorf("CE Content-Type %q is not supported", event.DataContentType())
		t.logger.Errorw("Bad Content-Type", zap.Error(err))
		return nil, err
	}

	localContext := ceContext{
		EventContextV1: event.Context.AsV1(),
		Extensions:     event.Context.AsV1().GetExtensions(),
	}

	localContextBytes, err := json.Marshal(localContext)
	if err != nil {
		t.logger.Errorw("Cannot encode CE context", zap.Error(err))
		return nil, fmt.Errorf("cannot encode CE context: %w", err)
	}

	// init indicates if we need to run initial step transformation
	var init = true
	var errs []error

	// Run init step such as load Pipeline variables first
	eventContext, err := contextPl.apply(localContextBytes, init)
	if err != nil {
		errs = append(errs, err)
	}
	eventPayload, err := dataPl.apply(event.Data(), init)
	if err != nil {
		errs = append(errs, err)
	}

	// CE Context transformation
	if eventContext, err = contextPl.apply(eventContext, !init); err != nil {
		errs = append(errs, err)
	}
	if err := json.Unmarshal(eventContext, &localContext); err != nil {
		t.logger.Errorw("Cannot decode CE new context", zap.Error(err))
		return nil, fmt.Errorf("cannot decode CE new context: %w", err)
	}
	event.Context = localContext
	for k, v := range localContext.Extensions {
		if err := event.Context.SetExtension(k, v); err != nil {
			t.logger.Errorw("Cannot set CE extension", zap.Error(err))
			return nil, fmt.Errorf("cannot set CE extension: %w", err)
		}
	}

	// CE Data transformation
	if eventPayload, err = dataPl.apply(eventPayload, !init); err != nil {
		errs = append(errs, err)
	}
	if err = event.SetData(cloudevents.ApplicationJSON, eventPayload); err != nil {
		t.logger.Errorw("Cannot set CE data", zap.Error(err))
		return nil, fmt.Errorf("cannot set CE data: %w", err)
	}
	// Failed transformation operations should not stop event flow
	// therefore, just log the errors
	if len(errs) != 0 {
		t.logger.Errorw("Event transformation errors", zap.Errors("errors", errs))
	}

	return &event, nil
}
