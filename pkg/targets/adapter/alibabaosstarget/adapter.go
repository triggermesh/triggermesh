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

package alibabaosstarget

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	cloudevents2 "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"
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
		cloudevents2.ReplierWithStaticResponseType(v1alpha1.EventTypeAlibabaOSSGenericResponse),
		cloudevents2.ReplierWithPayloadPolicy(cloudevents2.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	client, err := oss.New(env.Endpoint, env.AccessKeyID, env.AccessKeySecret)
	if err != nil {
		logger.Panicf("Error creating OSS client: %v", err)
	}

	return &ossAdapter{
		oClient: client,
		bucket:  env.Bucket,

		replier:  replier,
		ceClient: ceClient,
		logger:   logger,
	}
}

var _ pkgadapter.Adapter = (*ossAdapter)(nil)

type ossAdapter struct {
	oClient *oss.Client
	bucket  string

	replier  *cloudevents2.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

// Returns if stopCh is closed or Send() returns an error.
func (a *ossAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Alibaba OSS Adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *ossAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {

	bucket, err := a.oClient.Bucket(a.bucket)
	if err != nil {
		return a.replier.Error(&event, cloudevents2.ErrorCodeRequestParsing, err, nil)
	}

	if bucket == nil {
		return a.replier.Error(&event, cloudevents2.ErrorCodeRequestParsing, fmt.Errorf("no bucket returned"), nil)
	}

	if err = bucket.PutObject(event.ID(), bytes.NewReader(event.Data())); err != nil {
		return a.replier.Error(&event, cloudevents2.ErrorCodeAdapterProcess, err, nil)
	}

	return a.replier.Ok(&event, "ok")
}
