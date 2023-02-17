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

package splunktarget

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/ZachtimusPrime/Go-Splunk-HTTP/splunk/v2"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/metrics"
)

// SplunkClient is the interface that must be implemented by Splunk HEC
// clients.
type SplunkClient interface {
	NewEventWithTime(t time.Time, event interface{}, source, sourcetype, index string) *splunk.Event
	LogEvent(e *splunk.Event) error
}

// adapter implements the target's adapter.
type adapter struct {
	logger *zap.SugaredLogger

	ceClient cloudevents.Client
	spClient SplunkClient

	defaultIndex string

	sr *metrics.EventProcessingStatsReporter

	discardCEContext bool
}

var _ pkgadapter.Adapter = (*adapter)(nil)

// envConfig is a set parameters sourced from the environment for the target's adapter.
type envConfig struct {
	pkgadapter.EnvConfig

	HECEndpoint string `envconfig:"SPLUNK_HEC_ENDPOINT" required:"true"`
	HECToken    string `envconfig:"SPLUNK_HEC_TOKEN" required:"true"`
	Index       string `envconfig:"SPLUNK_INDEX"`

	SkipTLSVerify    bool `envconfig:"SPLUNK_SKIP_TLS_VERIFY"`
	DiscardCEContext bool `envconfig:"DISCARD_CE_CONTEXT" default:"false"`
}

// NewEnvConfig returns an accessor for the source's adapter envConfig.
func NewEnvConfig() pkgadapter.EnvConfigAccessor {
	return &envConfig{}
}

// https://docs.splunk.com/Documentation/Splunk/latest/RESTREF/RESTinput#services.2Fcollector.2Fevent.2F1.0
const eventURLPath = "/services/collector/event/1.0"

const httpTimeout = time.Second * 20

// NewTarget returns a constructor for the target's adapter.
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.SplunkTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envConfig)

	hecURL, err := url.Parse(env.HECEndpoint)
	if err != nil {
		logger.Panicw("Invalid HEC endpoint URL "+env.HECEndpoint, zap.Error(err))
	}

	return &adapter{
		logger: logger,

		ceClient: ceClient,
		spClient: newClient(*hecURL, env.HECToken, env.Index, hostname(envAcc), env.SkipTLSVerify),

		defaultIndex: env.Index,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),

		discardCEContext: env.DiscardCEContext,
	}
}

// newClient returns a Splunk HEC client.
func newClient(hecURL url.URL, hecToken, index, hostname string, skipTLSVerify bool) *splunk.Client {
	httpTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipTLSVerify,
		},
	}
	httpClient := &http.Client{
		Timeout:   httpTimeout,
		Transport: httpTransport,
	}

	if hecURL.Path == "" || hecURL.Path == "/" {
		hecURL.Path = eventURLPath
	}

	return &splunk.Client{
		HTTPClient: httpClient,
		URL:        hecURL.String(),
		Hostname:   hostname,
		Token:      hecToken,
		Index:      index,
	}
}

// hostname returns the host name to be included in Splunk events' metadata.
func hostname(env pkgadapter.EnvConfigAccessor) string {
	return "io.triggermesh.splunktarget." + env.GetNamespace() + "." + env.GetName()
}

// Start implements adapter.Adapter.
func (a *adapter) Start(ctx context.Context) error {
	errCh := make(chan error)
	go func() {
		errCh <- a.ceClient.StartReceiver(ctx, a.receive)
	}()

	return <-errCh
}

// receive implements the handler's receive logic.
func (a *adapter) receive(ctx context.Context, event cloudevents.Event) cloudevents.Result {
	a.logger.Debugw("Processing event", zap.Any("event", event))

	e := a.spClient.NewEventWithTime(
		event.Time(),
		event,
		event.Source(),
		event.Type(),
		a.defaultIndex,
	)
	if a.discardCEContext {
		e.Event = string(event.Data())
		if event.DataContentType() == cloudevents.ApplicationJSON {
			e.Event = json.RawMessage(event.Data())
		}
	}

	err := a.spClient.LogEvent(e)
	if err != nil {
		a.logger.Errorw("Failed to send event to HEC", zap.Error(err))
		return cloudevents.NewHTTPResult(a.extractHTTPStatus(err), "failed to send event to HEC: %s", err)
	}

	return cloudevents.ResultACK
}

// extractHTTPStatus attempts to extract the HTTP status code from the given
// error, returns "400 Bad Request" otherwise.
func (a *adapter) extractHTTPStatus(err error) int {
	if splunkErr, ok := err.(*splunk.EventCollectorResponse); ok {
		code, err := splunkErr.Code.HTTPCode()
		if err != nil {
			a.logger.Warnw("Couldn't determine HTTP status code", zap.Error(err))
			return http.StatusBadRequest
		}
		return code
	}

	return http.StatusBadRequest
}
