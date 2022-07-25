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

package alibabaosstarget

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	nkn "github.com/nknorg/nkn-sdk-go"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
	"github.com/triggermesh/triggermesh/pkg/targets/adapter/common"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.AlibabaOSSTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeAlibabaOSSGenericResponse),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	client, err := oss.New(env.Endpoint, env.AccessKeyID, env.AccessKeySecret)
	if err != nil {
		logger.Panicf("Error creating OSS client: %v", err)
	}

	var nknClient *nkn.Client
	var account *nkn.Account
	if env.EventTransportLayer == "NKN" {
		seed, err := hex.DecodeString(env.Seed)
		if err != nil {
			logger.Panicf("Error decoding seed from hex: %v", err)
		}

		account, err := nkn.NewAccount(seed)
		if err != nil {
			logger.Panicf("Error creating NKN account from seed: %v", err)
		}

		nknClient, err = nkn.NewClient(account, "any string", nil)
		if err != nil {
			logger.Panicf("Error creating NKN client: %v", err)
		}
	}

	return &ossAdapter{
		oClient: client,
		bucket:  env.Bucket,

		replier:        replier,
		ceClient:       ceClient,
		logger:         logger,
		transportLayer: env.EventTransportLayer,
		nknClient:      nknClient,
		nknAccount:     account,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*ossAdapter)(nil)

type ossAdapter struct {
	oClient *oss.Client
	bucket  string

	replier        *targetce.Replier
	ceClient       cloudevents.Client
	logger         *zap.SugaredLogger
	transportLayer string
	nknClient      *nkn.Client
	nknAccount     *nkn.Account

	sr *metrics.EventProcessingStatsReporter
}

// Returns if stopCh is closed or Send() returns an error.
func (a *ossAdapter) Start(ctx context.Context) error {
	a.logger.Infof("Starting Alibaba OSS Adapter with transport layer: %s", a.transportLayer)
	// Start the event transport layer
	if a.transportLayer == "CE" {

		return a.ceClient.StartReceiver(ctx, a.dispatch)
	}

	if a.transportLayer == "NKN" {
		return a.startNKN(ctx)
	}
	return nil
}

func (a *ossAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {

	bucket, err := a.oClient.Bucket(a.bucket)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}

	if bucket == nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, fmt.Errorf("no bucket returned"), nil)
	}

	if err = bucket.PutObject(event.ID(), bytes.NewReader(event.Data())); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	return a.replier.Ok(&event, "ok")
}

func (a *ossAdapter) startNKN(ctx context.Context) error {
	for {
		defer a.nknClient.Close()
		msg := <-a.nknClient.OnMessage.C
		a.logger.Debugf("Received NKN message: %s", msg)

		cloudEvent, err := common.ConvertNKNMessageToCloudevent(*msg)
		if err != nil {
			a.logger.Errorf("Error converting NKN message to CloudEvent: %v", err)
			break
		}

		e, r := a.dispatch(ctx, cloudEvent)

		if r != nil {
			a.logger.Errorf("Error dispatching: %v", r)
		} else {
			a.logger.Debugf("Dispatched: %s", e)
		}
	}
	return nil
}
