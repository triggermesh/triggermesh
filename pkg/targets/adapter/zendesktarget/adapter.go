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

package zendesktarget

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/nukosuke/go-zendesk/zendesk"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
)

const defaultTicketSubject = "TriggerMesh New Ticket Event"

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.ZendeskTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	return &zendeskAdapter{
		email:     env.Email,
		token:     env.Token,
		subdomain: env.Subdomain,
		subject:   env.Subject,

		ceClient: ceClient,
		logger:   logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*zendeskAdapter)(nil)

type zendeskAdapter struct {
	email     string
	token     string
	subdomain string
	subject   string
	zclient   *zendesk.Client

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

// Returns if stopCh is closed or Send() returns an error.
func (a *zendeskAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Zendesk adapter")

	newClient, err := zendesk.NewClient(nil)
	if err != nil {
		return err
	}

	a.zclient = newClient

	if err := a.zclient.SetSubdomain(a.subdomain); err != nil {
		return err
	}

	a.zclient.SetCredential(zendesk.NewAPITokenCredential(a.email, a.token))

	if err := a.ceClient.StartReceiver(ctx, a.dispatch); err != nil {
		return err
	}
	return nil
}

func (a *zendeskAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	switch typ := event.Type(); typ {
	case v1alpha1.EventTypeZendeskTicketCreate:
		ce, cr := a.createTicket(ctx, event)
		return ce, cr

	case v1alpha1.EventTypeZendeskTagCreate:
		ce, cr := a.tagTicket(ctx, event)
		return ce, cr

	default:
		res := fmt.Errorf("cannot process event with type %q", typ)
		return nil, res
	}
}

func (a *zendeskAdapter) createTicket(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	it := &TicketComment{}
	if err := event.DataAs(it); err != nil {
		res := fmt.Errorf("error processing incoming ticket: %w", err)
		return nil, res
	}

	ticket := &zendesk.Ticket{}

	// Assign the incoming ticket's subject to the new ticket.
	// If the new ticket subject is empty, use defaultTicketSubject
	ticket.Subject = it.Subject
	if ticket.Subject == "" {
		if a.subject != "" {
			ticket.Subject = a.subject
		} else {
			ticket.Subject = defaultTicketSubject
		}
	}

	if it.Body == "" {
		ticket.Comment.Body = event.String()
	} else {
		ticket.Comment.Body = it.Body
	}

	nT, err := a.zclient.CreateTicket(ctx, zendesk.Ticket(*ticket))
	if err != nil {
		res := fmt.Errorf("error creating ticket: %w", err)
		return nil, res
	}

	re, err := a.makeResponseEvent(ctx, nT)
	if err != nil {
		res := fmt.Errorf("error making response event: %w", err)
		return nil, res
	}

	a.logger.Info("Successfully created ticket #" + strconv.Itoa(int(nT.ID)))
	return re, cloudevents.ResultACK
}

func (a *zendeskAdapter) tagTicket(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	t := &TicketTag{}
	if err := event.DataAs(t); err != nil {
		res := fmt.Errorf("error processing incoming ticket: %w", err)
		return nil, res
	}

	if t.ID == 0 {
		res := fmt.Errorf("cannot update ticket tags without a ticket ID")
		return nil, res
	}

	if t.Tag == "" {
		res := fmt.Errorf("cannot update ticket tags with an empty tag")
		return nil, res
	}

	ot, err := a.zclient.GetTicket(ctx, t.ID)
	if err != nil {
		a.logger.Errorw("There was an error retrieving the requested Ticket for Tag updating")
	}

	nT, err := a.updateTag(ctx, ot, t.Tag)
	if err != nil {
		res := fmt.Errorf("failed to update tag: %w", err)
		return nil, res
	}

	re, err := a.makeResponseEvent(ctx, nT)
	if err != nil {
		res := fmt.Errorf("error making response event: %w", err)
		return nil, res
	}

	a.logger.Info("Successfully updated tag on ticket #" + strconv.Itoa(int(nT.ID)))
	return re, cloudevents.ResultACK
}

func (a *zendeskAdapter) updateTag(ctx context.Context, ticket zendesk.Ticket, tag string) (zendesk.Ticket, error) {
	var newTag = []string{tag}
	ticket.Tags = append(newTag, ticket.Tags...)

	uT, err := a.zclient.UpdateTicket(ctx, ticket.ID, ticket)
	if err != nil {
		a.logger.Errorw("An error has occurred updating the tag")
		return ticket, err
	}

	return uT, nil
}

func (a *zendeskAdapter) makeResponseEvent(ctx context.Context, ticket zendesk.Ticket) (*cloudevents.Event, error) {
	responseEvent := cloudevents.NewEvent(cloudevents.VersionV1)

	if err := responseEvent.SetData(cloudevents.ApplicationJSON, ticket); err != nil {
		a.logger.Errorw("Error generating response event", zap.Error(err))
		return nil, err
	}

	responseEvent.SetType("functions.zendesktargets.targets.triggermesh.io")
	responseEvent.SetSource(os.Getenv("NAMESPACE") + "/" + os.Getenv("NAME"))
	responseEvent.SetSubject("New Zendesk Ticket Created")

	return &responseEvent, nil
}
