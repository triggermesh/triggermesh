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

package elasticsearchtarget

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"

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

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeElasticsearchResponse),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	return &esAdapter{
		config:           env.GetElasticsearchConfig(),
		replier:          replier,
		index:            env.IndexName,
		discardCEContext: env.DiscardCEContext,
		ceClient:         ceClient,
		logger:           logger,
	}
}

var _ pkgadapter.Adapter = (*esAdapter)(nil)

type esAdapter struct {
	config *elasticsearch.Config
	client *elasticsearch.Client

	index string

	discardCEContext bool

	replier  *targetce.Replier
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
}

// Returns if stopCh is closed or Send() returns an error.
func (a *esAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Elasticsearch adapter")

	client, err := elasticsearch.NewClient(*a.config)
	if err != nil {
		return fmt.Errorf("failed to create Elasticsearch client: %v", err)
	}

	// Test proof the connection
	resp, err := client.Info()
	if err != nil {
		return fmt.Errorf("failed to retrieve info using Elasticsearch client: %v", err)
	}
	a.client = client
	if !resp.IsError() {
		info, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read info using Elasticsearch client: %v", err)
		}
		a.logger.Info("Connected to Elasticsearch: %s", string(info))
	}

	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *esAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	var data []byte

	if a.discardCEContext {
		data = event.Data()
	} else {
		jsonEvent, err := json.Marshal(event)
		if err != nil {
			return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
		}
		data = jsonEvent
	}

	req := esapi.IndexRequest{
		Index: a.index,
		Body:  bytes.NewReader(data),
	}

	res, err := req.Do(ctx, a.client)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)

	}
	defer res.Body.Close()
	if res.IsError() {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)

	}

	var resp map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeParseResponse, err, nil)

	}

	a.logger.Debug("Indexed CloudEvent: ", resp["result"])
	return a.replier.Ok(&event, res)
}
