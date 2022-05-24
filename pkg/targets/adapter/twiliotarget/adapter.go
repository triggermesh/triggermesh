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

package twiliotarget

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	twilio "github.com/kevinburke/twilio-go"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.TwilioTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeTwilioSMSSendResponse),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	// TODO custom port
	return &twilioAdapter{
		client:      twilio.NewClient(env.AccountSID, env.Token, nil),
		defaultFrom: env.PhoneFrom,
		defaultTo:   env.PhoneTo,

		replier:  replier,
		ceClient: ceClient,
		logger:   logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*twilioAdapter)(nil)

type twilioAdapter struct {
	client      *twilio.Client
	defaultFrom string
	defaultTo   string

	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

// Returns if stopCh is closed or Send() returns an error.
func (a *twilioAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Twilio adapter")

	if err := a.ceClient.StartReceiver(ctx, a.dispatch); err != nil {
		a.logger.Fatalw("Error listening to cloud events", zap.Error(err))
	}
	return nil
}

func (a *twilioAdapter) dispatch(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	if typ := event.Type(); typ != v1alpha1.EventTypeTwilioSMSSend {
		return a.replier.Error(&event, targetce.ErrorCodeEventContext, fmt.Errorf("event type %q is not supported", typ), nil)
	}

	sms := &SMSMessage{}
	if err := event.DataAs(sms); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}

	if sms.From == "" {
		sms.From = a.defaultFrom
	}
	if sms.To == "" {
		sms.To = a.defaultTo
	}

	if _, err := a.client.Messages.SendMessage(sms.From, sms.To, sms.Message, sms.MediaURLs); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	info := fmt.Sprint("message sent to: ", sms.To)
	a.logger.Debug(info)
	return a.replier.Ok(&event, info)
}
