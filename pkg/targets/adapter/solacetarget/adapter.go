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

package solacetarget

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"pack.ag/amqp"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/metrics"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.SolaceTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	connOption := amqp.ConnSASLAnonymous()

	if env.SASLEnable {
		connOption = amqp.ConnSASLPlain(env.Username, env.Password)
	}

	if env.TLSEnable {
		tlsCfg := &tls.Config{}
		if env.CA != "" {
			addCAConfig(tlsCfg, env.CA)
		}

		if env.ClientCert != "" || env.ClientKey != "" {
			if err := addTLSCerts(tlsCfg, env.ClientCert, env.ClientKey); err != nil {
				logger.Panicw("Could not parse the TLS Certificates", zap.Error(err))
			}
		}

		tlsCfg.InsecureSkipVerify = env.SkipVerify
		connOption = amqp.ConnTLSConfig(tlsCfg)
	}

	amqpOpts := []amqp.ConnOption{
		amqp.ConnIdleTimeout(0),
		connOption,
	}

	// Create amqp Client
	amqpClient, err := amqp.Dial(env.URL, amqpOpts...)
	if err != nil {
		logger.Panicw("Could not create the amqp client", zap.Error(err))
	}

	return &solaceAdapter{
		amqpClient: amqpClient,
		queueName:  env.QueueName,

		discardCEContext: env.DiscardCEContext,

		ceClient: ceClient,
		logger:   logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*solaceAdapter)(nil)

type solaceAdapter struct {
	amqpClient *amqp.Client
	amqpSender *amqp.Sender
	queueName  string

	discardCEContext bool

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

func (a *solaceAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Solace adapter")

	// Open a session
	session, err := a.amqpClient.NewSession()
	if err != nil {
		return err
	}

	// Create a sender
	a.amqpSender, err = session.NewSender(amqp.LinkTargetAddress(a.queueName))
	if err != nil {
		return err
	}

	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *solaceAdapter) dispatch(event cloudevents.Event) cloudevents.Result {
	ceTypeTag := metrics.TagEventType(event.Type())
	ceSrcTag := metrics.TagEventSource(event.Source())

	var msgVal []byte

	start := time.Now()
	defer func() {
		a.sr.ReportProcessingLatency(time.Since(start), ceTypeTag, ceSrcTag)
	}()

	if a.discardCEContext {
		msgVal = event.Data()
	} else {
		jsonEvent, err := json.Marshal(event)
		if err != nil {
			a.logger.Errorw("Error marshalling CloudEvent", zap.Error(err))
			a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
			return err
		}
		msgVal = jsonEvent
	}

	// Send message
	if err := a.amqpSender.Send(context.Background(), amqp.NewMessage(msgVal)); err != nil {
		a.logger.Errorw("Error producing Solace message", zap.String("msg", string(msgVal)), zap.Error(err))
		a.sr.ReportProcessingError(true, ceTypeTag, ceSrcTag)
		return err
	}

	return cloudevents.ResultACK
}

func addCAConfig(tlsConfig *tls.Config, caCert string) {
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(caCert))
	tlsConfig.RootCAs = caCertPool
}

func addTLSCerts(tlsConfig *tls.Config, clientCert, clientKey string) error {
	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err == nil {
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return err
}
