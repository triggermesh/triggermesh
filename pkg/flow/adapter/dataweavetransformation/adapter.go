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

package dataweavetransformation

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig
	// Spell defines the Dataweave spell to use on the incoming data at the event payload
	Spell string `envconfig:"DW_SPELL" required:"true"`
	// IncomingContentType defines the expected content type of the incoming data
	IncomingContentType string `envconfig:"CONTENT_TYPE" default:"application/json"`
	// BridgeIdentifier is the name of the bridge workflow this target is part of
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`
	// CloudEvents responses parametrization
	CloudEventPayloadPolicy string `envconfig:"EVENTS_PAYLOAD_POLICY" default:"error"`
	// Sink defines the target sink for the events. If no Sink is defined the
	// events are replied back to the sender.
	Sink string `envconfig:"K_SINK"`
}

// NewAdapter adapter implementation
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType("io.triggermesh.dataweavetransformation.error"),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	return &Adapter{
		spell:       env.Spell,
		contentType: env.IncomingContentType,

		sink:     env.Sink,
		replier:  replier,
		ceClient: ceClient,
		logger:   logger,
	}
}

var _ pkgadapter.Adapter = (*Adapter)(nil)

type Adapter struct {
	spell       string
	contentType string

	sink     string
	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

// Start is a blocking function and will return if an error occurs
// or the context is cancelled.
func (a *Adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting XMLToJSONTransformation Adapter")

	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *Adapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	// path := os.Getenv("KO_DATA_PATH") + "/static/dw"
	path := "dw"
	f := os.NewFile(0, "event.json")
	err := os.WriteFile("event.json", event.Data(), 0644)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "creating the file")
	}
	defer f.Close()

	cmd := path + ` -i payload event.json "` + a.spell + `"`
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "executing the spell")
	}

	output := string(out)
	cleaned := strings.ReplaceAll(output, "\n", "")
	event.SetData(a.contentType, cleaned)
	return &event, cloudevents.ResultACK
}
