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

package ocimetricssource

import (
	"context"
	"sync"
	"time"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/monitoring"

	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"
)

// OCIMetricsAPIHandler handles OCI Metrics.
type OCIMetricsAPIHandler interface {
	Start(ctx context.Context) error
}

type ociMetricsAPIHandler struct {
	provider    common.ConfigurationProvider
	client      monitoring.MonitoringClient
	interval    time.Duration
	metrics     []v1alpha1.OCIMetrics
	tenant      string
	ceClient    cloudevents.Client
	eventsource string
	context     context.Context

	logger *zap.SugaredLogger
}

// NewOCIMetricsAPIHandler returns a new instance of OCIMetricsAPIHandler.
func NewOCIMetricsAPIHandler(ceClient cloudevents.Client, env *envAccessor, eventsource string, logger *zap.SugaredLogger) OCIMetricsAPIHandler {
	interval, err := time.ParseDuration(env.PollingFrequency)
	if err != nil {
		logger.Panicw("cannot parse polling frequency", zap.Error(err))
	}

	// Ensure that the interval range is valid for the OCI Metrics API (min 1m max 1d)
	if interval < time.Minute || interval > time.Hour*24 {
		logger.Panic("interval is out of range")
	}

	provider := common.NewRawConfigurationProvider(env.TenantOCID, env.UserOCID, env.OracleRegion, env.OracleAPIKeyFingerprint, env.OracleAPIKey, &env.OracleAPIKeyPassphrase)

	monitoringClient, err := monitoring.NewMonitoringClientWithConfigurationProvider(provider)
	if err != nil {
		logger.Panicw("unable to create client", zap.Error(err))
	}

	return &ociMetricsAPIHandler{
		provider:    provider,
		ceClient:    ceClient,
		eventsource: eventsource,
		logger:      logger,
		interval:    interval,
		tenant:      env.TenantOCID,
		client:      monitoringClient,
		metrics:     env.Metrics,
	}
}

// Start starts the OCI Metrics events handler.
func (o *ociMetricsAPIHandler) Start(ctx context.Context) error {
	o.logger.Info("Starting OCI Metrics event handler with interval: ", o.interval)

	// Setup a timer for polling the metrics endpoint
	poll := time.NewTicker(o.interval)
	metricsCh := make(chan bool)
	defer poll.Stop()
	defer close(metricsCh)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	o.context = ctx

	// handle stop signals
	go func() {
		<-ctx.Done()
		o.logger.Debug("Shutdown signal received. Terminating")
		metricsCh <- true
		cancel()
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	// fire the initial metrics request, and then start polling
	go func() {
		prevTime := time.Now()

		for i := range o.metrics {
			o.collectMetrics(o.metrics[i], time.Now().Add(-o.interval), time.Now())
		}

		for {
			select {
			case <-metricsCh:
				wg.Done()
				return
			case t := <-poll.C:
				for i := range o.metrics {
					o.collectMetrics(o.metrics[i], prevTime, t)
				}
				prevTime = t
			}
		}
	}()

	wg.Wait()
	return nil
}

func (o *ociMetricsAPIHandler) collectMetrics(entry v1alpha1.OCIMetrics, startTime, endTime time.Time) {
	o.logger.Debug("Firing metrics")

	reqDetails := monitoring.SummarizeMetricsDataDetails{
		Namespace: &entry.MetricsNamespace,
		Query:     &entry.MetricsQuery,
		EndTime:   &common.SDKTime{Time: endTime},
		StartTime: &common.SDKTime{Time: startTime},
	}

	compartment := o.tenant
	if entry.Compartment != nil && *entry.Compartment != "" {
		compartment = *entry.Compartment
	}

	req := monitoring.SummarizeMetricsDataRequest{
		CompartmentId:               &compartment,
		SummarizeMetricsDataDetails: reqDetails,
	}

	response, err := o.client.SummarizeMetricsData(o.context, req)
	if err != nil {
		o.logger.Errorw("unable retrieving metrics", zap.Error(err))
	}

	event, err := o.cloudEventFromEventWrapper(entry.Name, &response)
	if err != nil {
		o.logger.Errorw("unable to package metrics", zap.Error(err))
	}

	if result := o.ceClient.Send(o.context, *event); !cloudevents.IsACK(result) {
		o.logger.Errorw("unable to send metrics", zap.Error(err))
	}
}

func (o *ociMetricsAPIHandler) cloudEventFromEventWrapper(subject string, response *monitoring.SummarizeMetricsDataResponse) (*cloudevents.Event, error) {
	event := cloudevents.NewEvent(cloudevents.VersionV1)

	// OpcRequestId may come back empty
	if response.OpcRequestId != nil {
		event.SetID(*response.OpcRequestId)
	}

	event.SetType(v1alpha1.OCIMetricsGenericEventType)
	event.SetSource(o.eventsource)
	event.SetSubject(subject)
	if err := event.SetData(cloudevents.ApplicationJSON, response.Items); err != nil {
		return nil, err
	}

	return &event, nil
}
