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

package twiliotarget

import (
	"context"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"

	twilio "github.com/kevinburke/twilio-go"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)

	// TODO custom port
	return &twilioAdapter{
		client:      twilio.NewClient(env.AccountSID, env.Token, nil),
		defaultFrom: env.PhoneFrom,
		defaultTo:   env.PhoneTo,
		ceClient:    ceClient,
		logger:      logger,
	}
}

var _ pkgadapter.Adapter = (*twilioAdapter)(nil)

type twilioAdapter struct {
	client      *twilio.Client
	defaultFrom string
	defaultTo   string

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

// Returns if stopCh is closed or Send() returns an error.
func (a *twilioAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Twilio adapter")

	if err := a.ceClient.StartReceiver(ctx, a.dispatch); err != nil {
		a.logger.Fatalw("Error listening to cloud events", zap.Error(err))
	}
	return nil
}

func (a *twilioAdapter) dispatch(event cloudevents.Event) cloudevents.Result {
	if typ := event.Type(); typ != v1alpha1.EventTypeTwilioSMSSend {
		return fmt.Errorf("cannot process event with type %q", typ)
	}

	sms := &SMSMessage{}
	if err := event.DataAs(sms); err != nil {
		return fmt.Errorf("error processing incoming event data: %w", err)
	}

	if sms.From == "" {
		sms.From = a.defaultFrom
	}
	if sms.To == "" {
		sms.To = a.defaultTo
	}

	if _, err := a.client.Messages.SendMessage(sms.From, sms.To, sms.Message, sms.MediaURLs); err != nil {
		return fmt.Errorf("error sending message: %w", err)
	}

	a.logger.Infof("Sent message: %s", event.String())
	return cloudevents.ResultACK
}
