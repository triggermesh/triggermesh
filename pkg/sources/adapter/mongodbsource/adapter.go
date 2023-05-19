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

package mongodbsource

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/sources/adapter/common/health"
)

const (
	resyncPeriod = 15 * time.Second
)

type envConfig struct {
	pkgadapter.EnvConfig

	MongoDBURI string `envconfig:"MONGODB_URI" required:"true"`
	Database   string `envconfig:"MONGODB_DATABASE" required:"true"`
	Collection string `envconfig:"MONGODB_COLLECTION" required:"true"`
}

type adapter struct {
	logger *zap.SugaredLogger
	mt     *pkgadapter.MetricTag

	mongoClient *mongo.Client
	ceClient    cloudevents.Client

	database   string
	collection string
}

// NewEnvConfig satisfies pkgadapter.EnvConfigConstructor.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.CloudEventsSourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	env := envAcc.(*envConfig)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(env.MongoDBURI))
	if err != nil {
		logger.Fatalw("Error connecting to MongoDB", zap.Error(err))
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		logger.Fatalw("Error pinging MongoDB", zap.Error(err))
	}

	return &adapter{
		logger: logger,
		mt:     mt,

		mongoClient: client,
		ceClient:    ceClient,

		database:   env.Database,
		collection: env.Collection,
	}
}

func (a *adapter) Start(ctx context.Context) error {
	go health.Start(ctx)

	health.MarkReady()
	a.logger.Info("Starting collection of MongoDB change events")
	ctx = pkgadapter.ContextWithMetricTag(ctx, a.mt)
	coll := a.mongoClient.Database(a.database).Collection(a.collection)
	cs, err := coll.Watch(ctx, mongo.Pipeline{})
	if err != nil {
		return fmt.Errorf("watching MongoDB collection: %w", err)
	}
	defer cs.Close(ctx)

	t := time.NewTicker(resyncPeriod)
	defer t.Stop()

	// Call processChanges initially
	if err := a.processChanges(ctx, cs); err != nil {
		a.logger.Errorw("Error processing changes", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-t.C:
			// Call processChanges when the ticker ticks
			if err := a.processChanges(ctx, cs); err != nil {
				a.logger.Errorw("Error processing changes", zap.Error(err))
			}
		}
	}
}

func (a *adapter) processChanges(ctx context.Context, cs *mongo.ChangeStream) error {
	for cs.Next(ctx) {
		var changeEvent bson.M
		if err := cs.Decode(&changeEvent); err != nil {
			a.logger.Errorw("Error decoding change event", zap.Error(err))
			continue
		}

		if err := a.processChangeEvent(ctx, changeEvent); err != nil {
			a.logger.Errorw("Error processing change event", zap.Error(err))
		}
	}

	if err := cs.Err(); err != nil {
		a.logger.Errorw("Error reading change events", zap.Error(err))
		return fmt.Errorf("reading change events: %w", err)
	}

	return nil
}

func (a *adapter) processChangeEvent(ctx context.Context, changeEvent bson.M) error {
	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetType(v1alpha1.MongoDBSourceEventType)
	event.SetSource(a.mt.Namespace + "/" + a.mt.Name)
	event.SetTime(time.Now())

	if err := event.SetData(cloudevents.ApplicationJSON, changeEvent); err != nil {
		return fmt.Errorf("setting event data: %w", err)
	}

	if result := a.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
		a.logger.Errorw("Failed to send event", zap.String("target", a.mt.Namespace+"/"+a.mt.Name), zap.Error(result))
		return fmt.Errorf("sending event: %w", result)
	}

	return nil
}

func (a *adapter) Stop() {
	// Close MongoDB client connection
	if err := a.mongoClient.Disconnect(context.Background()); err != nil {
		a.logger.Errorw("Error disconnecting from MongoDB", zap.Error(err))
	}
}
