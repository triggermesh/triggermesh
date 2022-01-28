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

package googlecloudstoragetarget

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)

	// Creates a client.
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON([]byte(env.Credentials)))
	if err != nil {
		logger.Panicf("Failed to create client: %v", err)
	}

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeGoogleCloudStorageResponse),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	return &googlecloudstorageAdapter{
		client: client,
		bucket: client.Bucket(env.BucketName),

		discardCEContext: env.DiscardCEContext,
		replier:          replier,
		ceClient:         ceClient,
		logger:           logger,
	}
}

var _ pkgadapter.Adapter = (*googlecloudstorageAdapter)(nil)

type googlecloudstorageAdapter struct {
	client *storage.Client
	bucket *storage.BucketHandle

	discardCEContext bool
	replier          *targetce.Replier
	ceClient         cloudevents.Client
	logger           *zap.SugaredLogger
}

// Returns if stopCh is closed or Send() returns an error.
func (a *googlecloudstorageAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Google Cloud Storage Adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *googlecloudstorageAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	if event.Type() == v1alpha1.EventTypeGoogleCloudStorageObjectInsert || a.discardCEContext {
		return a.insertObjectData(ctx, event)
	} else {
		return a.insertObject(ctx, event)
	}
}

func (a *googlecloudstorageAdapter) insertObjectData(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	ep := &EventPayload{}
	if err := event.DataAs(ep); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeCloudEventProcessing, err, "reading event data")
	}

	obj := a.bucket.Object(ep.FileName)
	w := obj.NewWriter(ctx)
	if _, err := w.Write(ep.Data); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "writing to storage")
	}

	if err := w.Close(); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "closing the writer")
	}

	return a.replier.Ok(&event, "ok")
}

func (a *googlecloudstorageAdapter) insertObject(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	d, err := json.Marshal(event)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeCloudEventProcessing, err, "reading event data")
	}

	obj := a.bucket.Object(event.ID() + ".json")
	w := obj.NewWriter(ctx)

	if _, err := w.Write(d); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "writing to storage")
	}
	if err := w.Close(); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, "closing the writer")
	}

	return a.replier.Ok(&event, "ok")
}
