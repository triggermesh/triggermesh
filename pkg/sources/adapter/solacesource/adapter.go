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

package solacesource

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"go.uber.org/zap"
	"pack.ag/amqp"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
	"github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
)

var _ pkgadapter.Adapter = (*solacesourceAdapter)(nil)

type solacesourceAdapter struct {
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
	mt       *pkgadapter.MetricTag

	amqpClient *amqp.Client
	queueName  string
}

// NewAdapter satisfies pkgadapter.AdapterConstructor.
func NewAdapter(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: sources.CloudEventsSourceResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	env := envAcc.(*envAccessor)

	amqpOpts := []amqp.ConnOption{
		amqp.ConnIdleTimeout(0),
	}

	if env.SASLEnable {
		amqpOpts = append(amqpOpts, amqp.ConnSASLPlain(env.Username, env.Password))
	} else {
		amqpOpts = append(amqpOpts, amqp.ConnSASLAnonymous())
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
		amqpOpts = append(amqpOpts, amqp.ConnTLSConfig(tlsCfg))
	}

	// Create amqp Client
	amqpClient, err := amqp.Dial(env.URL, amqpOpts...)
	if err != nil {
		logger.Panicw("Could not create the amqp client", zap.Error(err))
	}

	return &solacesourceAdapter{
		amqpClient: amqpClient,
		queueName:  env.QueueName,

		ceClient: ceClient,
		logger:   logger,
		mt:       mt,
	}
}

func (a *solacesourceAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Solace Source Adapter")

	// Open a session
	session, err := a.amqpClient.NewSession()
	if err != nil {
		return err
	}

	// Create a receiver
	receiver, err := session.NewReceiver(
		amqp.LinkSourceAddress(a.queueName),
	)
	if err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		receiver.Close(ctx)
		cancel()
	}()

	for {
		// Receive message
		msg, err := receiver.Receive(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			return err
		}

		err = a.emitEvent(ctx, msg)
		if err != nil {
			return err
		}

		err = msg.Accept()
		if err != nil {
			a.logger.Panicw("Error accepting messages", zap.Error(err))
		}
	}
}

func (a *solacesourceAdapter) emitEvent(ctx context.Context, msg *amqp.Message) error {
	event := cloudevents.NewEvent(cloudevents.VersionV1)
	event.SetType(v1alpha1.SolaceSourceEventType)
	event.SetSubject("solace/event")
	event.SetSource(a.queueName)
	event.SetID(msg.Properties.Subject)

	if err := event.SetData(cloudevents.ApplicationJSON, msg.GetData()); err != nil {
		return fmt.Errorf("failed to set event data: %w", err)
	}

	if result := a.ceClient.Send(ctx, event); !cloudevents.IsACK(result) {
		return result
	}

	return nil
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
