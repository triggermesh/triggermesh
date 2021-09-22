/*
Copyright (c) 2021 TriggerMesh Inc.

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

package sendgridtarget

import (
	"context"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	cloudevents2 "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
	"go.uber.org/zap"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)
	replier, err := cloudevents2.New(env.Component, logger.Named("replier"),
		cloudevents2.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		cloudevents2.ReplierWithStaticResponseType(v1alpha1.EventTypeSendGridEmailSendResponse),
		cloudevents2.ReplierWithPayloadPolicy(cloudevents2.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	return &sendGridAdapter{
		client:           sendgrid.NewSendClient(env.APIKey),
		defaultFromEmail: env.FromEmail,
		defaultToEmail:   env.ToEmail,
		defaultFromName:  env.FromName,
		defaultToName:    env.ToName,
		defaultMessage:   env.Message,
		defaultSubject:   env.Subject,
		replier:          replier,
		ceClient:         ceClient,
		logger:           logger,
	}

}

var _ pkgadapter.Adapter = (*sendGridAdapter)(nil)

type sendGridAdapter struct {
	client           *sendgrid.Client
	defaultFromEmail string
	defaultToEmail   string
	defaultFromName  string
	defaultToName    string
	defaultMessage   string
	defaultSubject   string

	replier  *cloudevents2.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

// Returns if stopCh is closed or Send() returns an error.
func (a *sendGridAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting SendGrid adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *sendGridAdapter) dispatch(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	switch typ := event.Type(); typ {
	case v1alpha1.EventTypeSendGridEmailSend:
		resp, err := a.sendEmail(event)
		if err != nil {
			return a.replier.Error(&event, cloudevents2.ErrorCodeEventContext, err, nil)
		}

		return a.replier.Ok(&event, resp)
	default:
		return a.replier.Error(&event, cloudevents2.ErrorCodeEventContext, fmt.Errorf("event type %q is not supported", event.Type()), nil)

	}
}

func (a *sendGridAdapter) sendEmail(event cloudevents.Event) (string, error) {
	email, err := a.defaultMessageData(event)
	if err != nil {
		return "", fmt.Errorf("Error processing incoming message")
	}

	from := mail.NewEmail(email.FromName, email.FromEmail)
	to := mail.NewEmail(email.ToName, email.ToEmail)
	plainTextContent := email.Message //plain text content is not being sent in the message body.
	htmlContent := "<strong>" + email.Message + "</strong>"
	message := mail.NewSingleEmail(from, email.Subject, to, plainTextContent, htmlContent)
	response, err := a.client.Send(message)
	if err != nil {
		return "", err
	}

	if response.StatusCode != 202 {
		return "", fmt.Errorf(response.Body)
	}

	a.logger.Infof("Sent message: %s", event.String())
	return response.Body, nil
}

// defaultMessageData populates our default data.
func (a *sendGridAdapter) defaultMessageData(e cloudevents.Event) (*EmailMessage, error) {
	m := &EmailMessage{}
	if err := e.DataAs(m); err != nil {
		return nil, err
	}
	if m.FromEmail == "" {
		m.FromEmail = a.defaultFromEmail
	}
	if m.ToEmail == "" {
		m.ToEmail = a.defaultToEmail
	}
	if m.FromName == "" {
		m.FromName = a.defaultFromName
	}
	if m.ToName == "" {
		m.ToName = a.defaultToName
	}
	if m.Message == "" {
		m.Message = e.String()
	}

	if m.Subject == "" {
		m.Subject = a.defaultSubject
	}
	return m, nil
}
