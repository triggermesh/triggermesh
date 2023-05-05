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

package httptarget

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/google/uuid"
	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/apis/targets/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/metrics"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.HTTPTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	u, err := url.Parse(env.URL)
	if err != nil {
		logger.Panicf("URL is not parseable: %v", err)
	}

	t := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: env.SkipVerify},
	}

	if env.CACertificate != "" {
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM([]byte(env.CACertificate)) {
			logger.Panicf("Failed adding certificate to pool: %s", env.CACertificate)
		}

		t.TLSClientConfig = &tls.Config{
			RootCAs: certPool,
		}
	}
	client := &http.Client{
		Transport: t,
	}

	if err = env.validateAuth(); err != nil {
		logger.Panic(err)
	}

	if env.isOAuth() {
		cfg := clientcredentials.Config{
			ClientID:     env.OAuthClientID,
			ClientSecret: env.OAuthClientSecret,
			TokenURL:     env.OAuthAuthTokenURL,
			Scopes:       env.OAuthScopes,
		}

		ctx = context.WithValue(ctx, oauth2.HTTPClient, client)
		client = cfg.Client(ctx)
	}

	return &httpAdapter{
		eventType:   env.EventType,
		eventSource: env.EventSource,

		url:               u,
		method:            env.Method,
		headers:           env.Headers,
		basicAuthUsername: env.BasicAuthUsername,
		basicAuthPassword: env.BasicAuthPassword,
		client:            client,

		ceClient: ceClient,
		logger:   logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*httpAdapter)(nil)

type httpAdapter struct {
	eventType   string
	eventSource string

	url               *url.URL
	method            string
	headers           map[string]string
	basicAuthUsername string
	basicAuthPassword string

	client *http.Client

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

// Returns if stopCh is closed or Send() returns an error.
func (a *httpAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting HTTP adapter")

	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *httpAdapter) dispatch(ctx context.Context, event cloudevents.Event) (*cloudevents.Event, cloudevents.Result) {
	rd := &RequestData{}
	if event.Type() != v1alpha1.EventTypeHTTPTargetRequest {
		rd.Body = event.Data()
	} else if err := event.DataAs(rd); err != nil {
		return nil, a.errorHTTPResult(http.StatusBadRequest, "Error processing incoming event data: %w", err)
	}

	u := *a.url

	if rd.Query != "" {
		kv, err := url.ParseQuery(rd.Query)
		if err != nil {
			return nil, a.errorHTTPResult(http.StatusBadRequest, "Error processing incoming event query string: %w", err)
		}

		values := u.Query()
		for k := range values {
			kv.Set(k, values.Get(k))
		}
		u.RawQuery = kv.Encode()
	}

	if rd.PathSuffix != "" {
		u.Path = path.Join(u.Path, rd.PathSuffix)
	}

	req, err := http.NewRequest(a.method, u.String(), bytes.NewBuffer(rd.Body))
	if err != nil {
		return nil, a.errorHTTPResult(http.StatusInternalServerError, "Could not create HTTP request: %w", err)
	}

	// apply spec headers to the request
	for k, v := range a.headers {
		req.Header.Set(k, v)
	}

	// apply request headers to the request. Might overwrite spec headers
	for k, v := range rd.Headers {
		req.Header.Set(k, v)
	}

	if a.basicAuthUsername != "" || a.basicAuthPassword != "" {
		req.SetBasicAuth(a.basicAuthUsername, a.basicAuthPassword)
	}

	res, err := a.client.Do(req)
	if err != nil {
		return nil, a.errorHTTPResult(http.StatusInternalServerError, "Error sending request: %w", err)
	}

	defer res.Body.Close()
	resb, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, a.errorHTTPResult(http.StatusInternalServerError, "Error reading response body: %w", err)
	}

	if res.StatusCode >= 400 {
		return nil, a.errorHTTPResult(res.StatusCode, "Received code %d from HTTP endpoint: %s", res.StatusCode, string(resb))
	}

	// build response event:
	// - ID is a new generated UUID
	// - content-type set to the one received at the HTTP response
	// - raw response data stored at the event data
	// - status code discarded since there will only be a response if the code is not an error
	// - Experimental: keeps the stateful headers if informed at the received event
	// - subject not informed

	out := cloudevents.NewEvent()
	if err := out.SetData(res.Header.Get("Content-Type"), resb); err != nil {
		return nil, a.errorHTTPResult(http.StatusInternalServerError, "Error setting response event data: %w", err)
	}

	ext := event.Context.GetExtensions()
	if stateID, ok := ext["statefulid"]; ok {
		if err := out.Context.SetExtension("statefulid", stateID); err != nil {
			return nil, a.errorHTTPResult(http.StatusInternalServerError, "Error setting stateful-id at event context: %w", err)
		}
	}

	if stateStep, ok := ext["statestep"]; ok {
		if err := out.Context.SetExtension("statestep", stateStep); err != nil {
			return nil, a.errorHTTPResult(http.StatusInternalServerError, "Error setting statestep at event context: %w", err)
		}
	}

	out.SetID(uuid.New().String())
	out.SetType(a.eventType)
	out.SetSource(a.eventSource)

	return &out, cloudevents.ResultACK
}

// errorResult given an error status code, writes an error log entry
// and returns a CloudEvents.Result
func (a *httpAdapter) errorHTTPResult(statusCode int, message string, args ...interface{}) cloudevents.Result {
	r := cloudevents.NewHTTPResult(statusCode, message, args...)
	a.logger.Error(r.Error())
	return r
}
