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

package infratarget

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	"github.com/triggermesh/triggermesh/pkg/targets/adapter/infratarget/vm"
	jsvm "github.com/triggermesh/triggermesh/pkg/targets/adapter/infratarget/vm/javascript"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.InfraTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	adapter := &infraAdapter{
		ceClient:           ceClient,
		logger:             logger,
		typeLoopProtection: env.TypeLoopProtection,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}

	if env.ScriptCode != "" {
		adapter.vm = jsvm.New(env.ScriptCode, time.Duration(env.ScriptTimeout)*time.Millisecond, logger.Named("vm"))
	}

	switch env.StateHeadersPolicy {
	case "ensure":
		adapter.preProcessHeaders = ensureStateHeaders(env.StateBridge)
		fallthrough
	case "propagate":
		adapter.postProcessHeaders = propagateStateHeaders
	}

	return adapter
}

var _ pkgadapter.Adapter = (*infraAdapter)(nil)

type infraAdapter struct {
	vm                 vm.InfraVM
	preProcessHeaders  preProcessHeaders
	postProcessHeaders postProcessHeaders
	typeLoopProtection bool

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

// Returns if stopCh is closed or Send() returns an error.
func (a *infraAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Infra adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *infraAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	var out *cloudevents.Event
	var err error

	// Preprocess headers will modify incoming event.
	// This is the chance to add new headers.
	if a.preProcessHeaders != nil {
		if err = a.preProcessHeaders(&event); err != nil {
			r := cloudevents.NewHTTPResult(http.StatusInternalServerError, "Pre processing headers: %w", err)
			a.logger.Errorw("Pre processing headers error", zap.Error(r))
			return nil, r
		}
	}

	// VM execution receives preprocessed event as input but does not modify it,
	// instead it returns a new output event.
	if a.vm != nil {
		out, err = a.vm.Exec(&event)
		if err != nil {
			r := cloudevents.NewHTTPResult(http.StatusInternalServerError, "Error executing script: %w", err)
			a.logger.Errorw("Error executing script", zap.Error(r))
			return nil, r
		}
	} else {
		out = &event
	}

	// if no CloudEvent is produced there is no need for post-processing
	if out == nil {
		return nil, cloudevents.ResultACK
	}

	// if event type loop protection is enabled, make sure the output type
	// does not match the incoming type.
	if a.typeLoopProtection {
		if event.Type() == out.Type() {
			r := cloudevents.NewHTTPResult(http.StatusInternalServerError, "incoming and outgoing CloudEvents have the same type %q. Skipping", event.Type())
			a.logger.Errorw("CE type error", zap.Error(r))
			return nil, r
		}
	}

	// Postprocess headers modifies the output event using the preprocessed
	// incoming event. Missing headers from input event might be copied to the
	// output as part of this process.
	if a.postProcessHeaders != nil && out != nil {
		if err = a.postProcessHeaders(&event, out); err != nil {
			r := cloudevents.NewHTTPResult(http.StatusInternalServerError, "Post processing headers: %w", err)
			a.logger.Errorw("Post processing headers", zap.Error(r))
			return nil, r
		}
	}

	return out, cloudevents.ResultACK
}

type postProcessHeaders func(in, out *cloudevents.Event) error
type preProcessHeaders func(event *cloudevents.Event) error

// propagateStateHeaders copies missing state headers from incoming
// to outgoing CloudEvent
func propagateStateHeaders(in, out *cloudevents.Event) error {
	extout := out.Context.GetExtensions()
	extin := in.Context.GetExtensions()

	var val interface{}

	// bridge value is shared among all instances of the event
	// workflow created from this bridge.
	if _, ok := extout["statefulbridge"]; !ok {
		if val, ok = extin["statefulbridge"]; ok {
			if err := out.Context.SetExtension("statefulbridge", val); err != nil {
				return fmt.Errorf("error setting statefulbridge extension: %w", err)
			}
		}
	}

	// statefulid value is unique per event workflow instance, if
	// empty create a new one.
	if _, ok := extout["statefulid"]; !ok {
		if val, ok = extin["statefulid"]; ok {
			if err := out.Context.SetExtension("statefulid", val); err != nil {
				return fmt.Errorf("error setting statefulid extension: %w", err)
			}
		}
	}

	// statestep is a free value that can be used to track the
	// running workflow when needed.
	if _, ok := extout["statestep"]; !ok {
		if val, ok = extin["statestep"]; ok {
			if err := out.Context.SetExtension("statestep", val); err != nil {
				return fmt.Errorf("error setting statefulid extension: %w", err)
			}
		}
	}

	return nil
}

// ensureStateHeaders given an event and a bridge workflow, sets the
// stateful headers if they don't exists.
func ensureStateHeaders(bridge string) preProcessHeaders {
	return func(event *cloudevents.Event) error {
		// Add defaults to missing stateful headers
		ext := event.Context.GetExtensions()

		if _, ok := ext["statefulbridge"]; !ok {
			if err := event.Context.SetExtension("statefulbridge", bridge); err != nil {
				return fmt.Errorf("error ensuring statefulbridge extension: %w", err)
			}
		}

		if _, ok := ext["statefulid"]; !ok {
			if err := event.Context.SetExtension("statefulid", uuid.New().String()); err != nil {
				return fmt.Errorf("error ensuring statefulid extension: %w", err)
			}
		}
		return nil
	}
}
