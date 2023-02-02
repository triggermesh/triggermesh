/*
Copyright 2023 TriggerMesh Inc.

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

// Package mongodbtarget implements an adapter that connects to a MongoDB database
// and allows a user to insert, query, and update documents via cloudevents.
package mongodbtarget

import (
	"context"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

// NewTarget returns the adapter implementation.
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)
	metrics.MustRegisterEventProcessingStatsView()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(env.ServerURL))
	if err != nil {
		logger.Panicw("error connecting to mongodb", zap.Error(err))
		return nil
	}

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeMongoDBStaticResponse),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.MongoDBTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	return &adapter{
		mclient:           client,
		defaultDatabase:   env.DefaultDatabase,
		defaultCollection: env.DefaultCollection,

		replier:  replier,
		ceClient: ceClient,
		logger:   logger,
		sr:       metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*adapter)(nil)

type adapter struct {
	mclient           *mongo.Client
	defaultDatabase   string
	defaultCollection string

	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
	sr       *metrics.EventProcessingStatsReporter
}

func (a *adapter) Start(ctx context.Context) error {
	a.logger.Info("Starting MongoDB adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *adapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	ceTypeTag := metrics.TagEventType(event.Type())
	ceSrcTag := metrics.TagEventSource(event.Source())
	a.logger.Debug("Processing event:" + event.Type())
	start := time.Now()
	defer func() {
		a.sr.ReportProcessingLatency(time.Since(start), ceTypeTag, ceSrcTag)
	}()

	switch typ := event.Type(); typ {
	case v1alpha1.EventTypeMongoDBInsert:
		if err := a.insert(event, ctx); err != nil {
			a.logger.Errorw("invoking .insert: ", zap.Error(err))
			a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
	case v1alpha1.EventTypeMongoDBQueryKV:
		resp, err := a.kvQuery(event, ctx)
		if err != nil {
			a.logger.Errorw("invoking .query.kv: ", zap.Error(err))
			a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
		return resp, nil
	case v1alpha1.EventTypeMongoDBUpdate:
		if err := a.update(event, ctx); err != nil {
			a.logger.Errorw("invoking .update: ", zap.Error(err))
			a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
	default:
		if err := a.insertFromConfig(event, ctx); err != nil {
			a.logger.Errorw("invoking arbirary insert: ", zap.Error(err))
			a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
			return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
		}
	}
	return a.replier.Ok(&event, "ok")
}

// insertFromConfig inserts a mongodb document using the provided collection and database
// at the spec configuration.
func (a *adapter) insertFromConfig(e cloudevents.Event, ctx context.Context) error {
	var payload map[string]interface{}
	if err := e.DataAs(&payload); err != nil {
		return err
	}

	collection := a.mclient.Database(a.defaultDatabase).Collection(a.defaultCollection)
	_, err := collection.InsertOne(ctx, payload)
	if err != nil {
		return err
	}
	return nil
}

// kvQuery queries a mongodb collection for a specific key/value pair.
func (a *adapter) kvQuery(e cloudevents.Event, ctx context.Context) (*cloudevents.Event, error) {
	qpd := &QueryPayload{}
	if err := e.DataAs(qpd); err != nil {
		return nil, err
	}
	col := a.defaultCollection
	db := a.defaultDatabase
	if qpd.Collection != "" {
		col = qpd.Collection
	}
	if qpd.Database != "" {
		db = qpd.Database
	}

	collection := a.mclient.Database(db).Collection(col)
	filterCursor, err := collection.Find(ctx, bson.M{qpd.Key: qpd.Value})
	if err != nil {
		return nil, err
	}

	var itemsFiltered []bson.M
	if err = filterCursor.All(ctx, &itemsFiltered); err != nil {
		return nil, err
	}

	responseEvent := cloudevents.NewEvent(cloudevents.VersionV1)
	err = responseEvent.SetData(cloudevents.ApplicationJSON, itemsFiltered)
	if err != nil {
		return nil, err
	}

	responseEvent.SetType(v1alpha1.EventTypeMongoDBQueryResponse)
	responseEvent.SetSource(fmt.Sprintf("%s-%s", db, col))
	responseEvent.SetSubject("query-result")
	responseEvent.SetDataContentType(cloudevents.ApplicationJSON)
	return &responseEvent, nil
}

// insert inserts a document into a mongodb collection.
func (a *adapter) insert(e cloudevents.Event, ctx context.Context) error {
	ipd := &InsertPayload{}
	if err := e.DataAs(ipd); err != nil {
		return err
	}
	col := a.defaultCollection
	db := a.defaultDatabase

	// override default collection and database if specified in the event
	if ipd.Collection != "" {
		col = ipd.Collection
	}
	if ipd.Database != "" {
		db = ipd.Database
	}

	collection := a.mclient.Database(db).Collection(col)
	if ipd.Document != nil {
		_, err := collection.InsertOne(ctx, ipd.Document)
		if err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("no data to insert")
}

// update updates a document in a mongodb collection.
func (a *adapter) update(e cloudevents.Event, ctx context.Context) error {
	up := &UpdatePayload{}
	if err := e.DataAs(up); err != nil {
		return err
	}
	col := a.defaultCollection
	db := a.defaultDatabase
	if up.Collection != "" {
		col = up.Collection
	}
	if up.Database != "" {
		db = up.Database
	}

	collection := a.mclient.Database(db).Collection(col)
	_, err := collection.UpdateOne(
		ctx,
		bson.M{up.SearchKey: up.SearchValue},
		bson.D{{Key: "$set", Value: bson.D{{Key: up.UpdateKey, Value: up.UpdateValue}}}},
	)
	if err != nil {
		return err
	}

	return nil
}
