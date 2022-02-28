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

package opentelemetrytarget

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	// "This package is no longer supported. Use the go.opentelemetry.io/otel/exporters/otlp/otlpmetric exporter as
	// a replacement to send data to a collector which can then export with its PRW exporter."
	"go.opentelemetry.io/contrib/exporters/metric/cortex" //nolint:staticcheck

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/number"
	"go.opentelemetry.io/otel/metric/sdkapi"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

var _ pkgadapter.Adapter = (*cortexAdapter)(nil)

// instrumentRef is a reference to an instrument descriptor and its implementation.
type instrumentRef struct {
	descriptor sdkapi.Descriptor
	sync       sdkapi.SyncImpl
}

type opentelemetryAdapter struct {
	instruments map[string]map[string]*instrumentRef

	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

type cortexAdapter struct {
	opentelemetryAdapter

	cortexConfig *cortex.Config
}

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	if len(env.Instruments) == 0 {
		logger.Panic("No instruments present")
	}

	// instruments structure is a map nested with two keys.
	//
	// - Instrument name: this will usually be unique, but it could
	//   happen that two different Instrument kinds share the same name
	// - Instrument kind: there will usually be only one kind per name
	// 	 but that is not guaranteed.
	//
	// We use it to keep the registered set of instruments in order to
	// match with the incoming CloudEvent requests.
	instruments := map[string]map[string]*instrumentRef{}
	for _, i := range env.Instruments {
		if _, ok := instruments[i.Name]; !ok {
			instruments[i.Name] = map[string]*instrumentRef{}
		}
		instruments[i.Name][i.Instrument] = &instrumentRef{descriptor: i.Descriptor}
	}

	ccfg := &cortex.Config{
		Endpoint:      env.CortexEndpoint,
		BearerToken:   env.CortexBearerToken,
		RemoteTimeout: env.CortexRemoteTimeout,
		PushInterval:  env.CortexPushInterval,
	}

	if err = ccfg.Validate(); err != nil {
		logger.Panicf("Error validating Cortex configuration: %v", err)
	}

	return &cortexAdapter{
		cortexConfig: ccfg,
		opentelemetryAdapter: opentelemetryAdapter{
			instruments: instruments,

			replier:  replier,
			ceClient: ceClient,
			logger:   logger,
		},
	}
}

// Start is a blocking function and will return if an error occurs
// or the context is cancelled.
func (a *cortexAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Cortex adapter")

	cortexctl, err := cortex.InstallNewPipeline(
		*a.cortexConfig,
		controller.WithCollectPeriod(a.cortexConfig.PushInterval),
	)
	if err != nil {
		return fmt.Errorf("failed to create Cortex controller: %v", err)
	}

	defer func() {
		if err := cortexctl.Stop(ctx); err != nil {
			// Warning only, this will be most of the time a context
			// cancellation error, which is not an issue.
			a.logger.Warnw("Error stopping Cortex adapter", zap.Error(err))
		}
	}()

	meter := cortexctl.Meter("TriggerMesh")

	// Iterate over all instruments, create their instances and
	// store the link back to the instruments map.
	for name, kindm := range a.instruments {
		for kind, i := range kindm {

			if i.descriptor.InstrumentKind().Synchronous() {
				i.sync, err = meter.MeterImpl().NewSyncInstrument(i.descriptor)
				if err != nil {
					return fmt.Errorf("failed to create sync instrument: %v", err)
				}
				continue
			}

			if i.descriptor.InstrumentKind().Asynchronous() {
				return fmt.Errorf("async instrument %s/%s not supported", name, kind)
			}
			return fmt.Errorf("cannot determine if the instrument %s/%s is sync or async", name, kind)
		}
	}

	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *opentelemetryAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {

	if typ := event.Type(); typ != v1alpha1.EventTypeOpenTelemetryMetricsPush {
		return a.replier.Error(&event, targetce.ErrorCodeEventContext, fmt.Errorf("event type %q is not supported", typ), nil)
	}

	ms := []Measure{}
	if err := event.DataAs(&ms); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}

	errs := []error{}
	for i := range ms {
		if err := a.processSingleMeasure(ctx, ms[i]); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 {
		return a.replier.Ack()
	}

	kerrs := kerrors.NewAggregate(errs)
	return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, kerrs, nil)
}

func (a *opentelemetryAdapter) processSingleMeasure(ctx context.Context, m Measure) error {
	attrs := make([]attribute.KeyValue, len(m.Attributes))
	for i := range m.Attributes {
		attr, err := m.Attributes[i].ParseAttribute()
		if err != nil {
			return err
		}
		attrs[i] = *attr
	}

	// Match the measure with an instrument
	kindm, ok := a.instruments[m.Name]
	if !ok {
		return fmt.Errorf("instrument %q has not been configured", m.Name)
	}

	// Look for matching Kind or defaulting if there is only
	// one for the instrument.
	var ir *instrumentRef
	switch {
	case m.Kind != "":
		if ir, ok = kindm[m.Kind]; !ok {
			return fmt.Errorf("undefined kind %q for instrument %q", m.Kind, m.Name)
		}

	case len(kindm) == 1:
		for _, v := range kindm {
			ir = v
		}

	default:
		return fmt.Errorf("instrument %q has multiple kinds. Measure did not specify one", m.Name)
	}

	// Parse value according to the instrument number kind
	var value number.Number
	switch ir.descriptor.NumberKind() {
	case number.Int64Kind:
		var v int64
		if err := json.Unmarshal(m.Value, &v); err != nil {
			return fmt.Errorf("value %v cannot be parsed as int64: %w", m.Name, err)
		}
		value = number.NewInt64Number(v)

	case number.Float64Kind:
		var v float64
		if err := json.Unmarshal(m.Value, &v); err != nil {
			return fmt.Errorf("value %v cannot be parsed as float64: %w", m.Name, err)
		}
		value = number.NewFloat64Number(v)
	}

	switch {
	case ir.descriptor.InstrumentKind().Synchronous():
		ir.sync.RecordOne(ctx, value, attrs)

	case ir.descriptor.InstrumentKind().Asynchronous():
		return errors.New("async instrument is not supported")

	default:
		return errors.New("instrument is not sync nor async")
	}
	return nil
}
