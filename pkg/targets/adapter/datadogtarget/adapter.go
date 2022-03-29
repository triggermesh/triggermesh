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

package datadogtarget

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	targetce "github.com/triggermesh/triggermesh/pkg/targets/adapter/cloudevents"
)

const (
	apiBaseURL     = "https://api.datadoghq.com"
	logsAPIBaseURL = "https://http-intake.logs.datadoghq.com"
	apiKeyHeader   = "DD-API-KEY"
)

const (
	contentTypeHeader = "Content-Type"
	contentTypeJSON   = "application/json"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	env := envAcc.(*envAccessor)
	logger := logging.FromContext(ctx)

	replier, err := targetce.New(env.Component, logger.Named("replier"),
		targetce.ReplierWithStatefulHeaders(env.BridgeIdentifier),
		targetce.ReplierWithStaticResponseType(v1alpha1.EventTypeDatadogResponse),
		targetce.ReplierWithPayloadPolicy(targetce.PayloadPolicy(env.CloudEventPayloadPolicy)))
	if err != nil {
		logger.Panicf("Error creating CloudEvents replier: %v", err)
	}

	return &datadogAdapter{
		apiKey: env.APIKey,

		replier:    replier,
		httpClient: http.DefaultClient,
		ceClient:   ceClient,
		logger:     logger,
	}
}

var _ pkgadapter.Adapter = (*datadogAdapter)(nil)

type datadogAdapter struct {
	apiKey string

	replier    *targetce.Replier
	httpClient *http.Client
	ceClient   cloudevents.Client
	logger     *zap.SugaredLogger
}

// Returns if stopCh is closed or Send() returns an error.
func (a *datadogAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Datadog adapter")
	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *datadogAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	switch typ := event.Type(); typ {
	case v1alpha1.EventTypeDatadogMetric:
		return a.postMetric(event)
	case v1alpha1.EventTypeDatadogEvent:
		return a.postEvent(event)
	case v1alpha1.EventTypeDatadogLog:
		return a.postLog(event)
	default:
		return a.replier.Error(&event, targetce.ErrorCodeEventContext, fmt.Errorf("event type %q is not supported", typ), nil)
	}
}

func (a *datadogAdapter) postLog(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	if err := event.DataAs(&LogData{}); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}

	request, err := newLogsAPIRequest("/v1/input", a.apiKey, event.Data())
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)

	}

	res, err := a.httpClient.Do(request)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeParseResponse, err, nil)
	}

	if res.StatusCode != http.StatusOK {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess,
			fmt.Errorf("received HTTP code %d", res.StatusCode),
			map[string]string{"body": string(resBody)})
	}

	return a.replier.Ok(&event, resBody)
}

func (a *datadogAdapter) postEvent(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	if err := event.DataAs(&EventData{}); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}

	request, err := newAPIRequest("/api/v1/events", a.apiKey, event.Data())
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)

	}

	res, err := a.httpClient.Do(request)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeParseResponse, err, nil)
	}

	if res.StatusCode != http.StatusAccepted {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess,
			fmt.Errorf("received HTTP code %d", res.StatusCode),
			map[string]string{"body": string(resBody)})
	}

	return a.replier.Ok(&event, resBody)
}

func (a *datadogAdapter) postMetric(event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	if err := event.DataAs(&MetricData{}); err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeRequestParsing, err, nil)
	}

	request, err := newAPIRequest("/api/v1/series", a.apiKey, event.Data())
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	res, err := a.httpClient.Do(request)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess, err, nil)
	}

	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return a.replier.Error(&event, targetce.ErrorCodeParseResponse, err, nil)
	}

	if res.StatusCode != http.StatusAccepted {
		return a.replier.Error(&event, targetce.ErrorCodeAdapterProcess,
			fmt.Errorf("received HTTP code %d", res.StatusCode),
			map[string]string{"body": string(resBody)})
	}

	return a.replier.Ok(&event, resBody)
}

// newAPIRequest returns a POST http.Request that is ready to send to the Datadog general-purpose API.
func newAPIRequest(path, apiKey string, body []byte) (*http.Request, error) {
	return newAPIRequestWithHost(apiBaseURL, path, apiKey, body)
}

// newLogsAPIRequest returns a POST http.Request that is ready to send to the Datadog logs API.
func newLogsAPIRequest(path, apiKey string, body []byte) (*http.Request, error) {
	return newAPIRequestWithHost(logsAPIBaseURL, path, apiKey, body)
}

// newAPIRequestWithHost returns a POST http.Request that is ready to send to the Datadog API.
func newAPIRequestWithHost(host, path, apiKey string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodPost, host+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set(contentTypeHeader, contentTypeJSON)
	req.Header.Set(apiKeyHeader, apiKey)

	return req, nil
}
