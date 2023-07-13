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

package googlecloudfirestoretarget

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.GoogleCloudFirestoreTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	opts := make([]option.ClientOption, 0)
	if env.ServiceAccountKey != nil {
		opts = append(opts, option.WithCredentialsJSON(env.ServiceAccountKey))
	}

	// Creates a client.
	client, err := firestore.NewClient(ctx, env.ProjectID, opts...)
	if err != nil {
		logger.Panicf("Failed to create client: %v", err)
	}

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeGoogleCloudFirestoreWriteResponse),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	return &googlecloudFirestoreAdapter{
		client:            client,
		defaultCollection: env.DefaultCollection,
		discardCEContext:  env.DiscardCEContext,

		replier:  replier,
		ceClient: ceClient,
		logger:   logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*googlecloudFirestoreAdapter)(nil)

type googlecloudFirestoreAdapter struct {
	client *firestore.Client

	defaultCollection string
	discardCEContext  bool

	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

// Returns if stopCh is closed or Send() returns an error.
func (a *googlecloudFirestoreAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Google Cloud Firestore Adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *googlecloudFirestoreAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {

	switch typ := event.Type(); typ {
	case v1alpha1.EventTypeGoogleCloudFirestoreWrite:
		return a.insertObject(ctx, event)
	case v1alpha1.EventTypeGoogleCloudFirestoreQueryTables:
		return a.queryTables(ctx, event)
	case v1alpha1.EventTypeGoogleCloudFirestoreQueryTable:
		return a.queryTable(ctx, event)
	default:
		return a.instertArbitraryObject(ctx, event)
	}
}

func (a *googlecloudFirestoreAdapter) insertObject(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	ep := &EventPayload{}
	if err := event.DataAs(ep); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}
	if ep.Collection == "" {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation, fmt.Errorf("must include a 'collection' attribute in the payload"), nil)
	}

	if ep.Document == "" {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation, fmt.Errorf("must include a 'document' attribute in the payload"), nil)
	}

	if ep.Data == nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation, fmt.Errorf("must include a 'data' attribute in the payload"), nil)
	}

	col := a.client.Collection(ep.Collection)
	if col == nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation, fmt.Errorf("collection '%s' not found", ep.Collection), nil)
	}

	doc := col.Doc(ep.Document)

	wr, err := doc.Create(ctx, ep.Data)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	return a.replier.Ok(&event, wr)
}

func (a *googlecloudFirestoreAdapter) instertArbitraryObject(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	var eventJSONMap map[string]interface{}

	if a.discardCEContext {
		if err := event.DataAs(&eventJSONMap); err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeParseResponse, err, nil)
		}
	} else {
		b, err := json.Marshal(event)
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeParseResponse, err, nil)
		}
		if err := json.Unmarshal(b, &eventJSONMap); err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeParseResponse, err, nil)
		}
	}

	col := a.client.Collection(a.defaultCollection)
	if col == nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, fmt.Errorf("a default collection was not set in the spec"), nil)
	}
	doc := col.Doc(event.ID())

	wr, err := doc.Create(ctx, eventJSONMap)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	return a.replier.Ok(&event, wr)
}

func (a *googlecloudFirestoreAdapter) queryTables(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	ep := &EventPayload{}
	var d []interface{}

	if err := event.DataAs(ep); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}

	if ep.Collection == "" {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation, fmt.Errorf("element 'collection' is mandatory in the payload"), nil)
	}

	iter := a.client.Collection(ep.Collection).Documents(ctx)
	defer iter.Stop()
	if iter == nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation, fmt.Errorf("no objects found"), nil)
	}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeParseResponse, err, nil)
		}

		data := doc.Data()
		d = append(d, data)
	}

	return a.replier.Ok(&event, d)
}

func (a *googlecloudFirestoreAdapter) queryTable(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	ep := &EventPayload{}
	if err := event.DataAs(ep); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}

	if ep.Collection == "" {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation, fmt.Errorf("must include a 'collection' attribute in the payload"), nil)
	}

	if ep.Document == "" {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation, fmt.Errorf("must include a 'document' attribute in the payload"), nil)
	}

	dsnap, err := a.client.Collection(ep.Collection).Doc(ep.Document).Get(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}

	if dsnap.Data() == nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestValidation, fmt.Errorf("no data found"), nil)
	}

	d := dsnap.Data()

	return a.replier.Ok(&event, d)
}
