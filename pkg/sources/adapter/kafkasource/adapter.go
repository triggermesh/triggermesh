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

package kafkasource

import (
	"context"
	"crypto/tls"
	"crypto/x509"

	"go.uber.org/zap"

	"github.com/Shopify/sarama"
	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/triggermesh/triggermesh/pkg/apis/sources"
)

var _ pkgadapter.Adapter = (*kafkasourceAdapter)(nil)

type kafkasourceAdapter struct {
	ceClient cloudevents.Client
	logger   *zap.SugaredLogger
	mt       *pkgadapter.MetricTag

	kafkaClient sarama.ConsumerGroup
	topics      []string
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

	var err error

	config := sarama.NewConfig()
	tlsCfg := &tls.Config{}

	if env.SASLEnable {
		config.Net.SASL.Enable = env.SASLEnable
		config.Net.SASL.Mechanism = sarama.SASLMechanism(env.SecurityMechanisms)
		config.Net.SASL.User = env.Username
		config.Net.SASL.Password = env.Password
	}

	if env.TLSEnable {
		config.Net.TLS.Enable = env.TLSEnable
		tlsCfg, err = newTLSCertificatesConfig(tlsCfg, env.ClientCert, env.ClientKey)
		if err != nil {
			logger.Panicw("Could not create the TLS Certificates Config", err)
		}
		tlsCfg = newTLSRootCAConfig(tlsCfg, env.CA)
		config.Net.TLS.Config = tlsCfg
		config.Net.TLS.Config.InsecureSkipVerify = env.SkipVerify
	}

	if env.SecurityMechanisms == "GSSAPI" {
		kerberosConfig := sarama.GSSAPIConfig{
			KerberosConfigPath: env.KerberosConfigPath,
			ServiceName:        env.KerberosServiceName,
			Username:           env.KerberosUsername,
			Password:           env.KerberosPassword,
			Realm:              env.KerberosRealm,
			DisablePAFXFAST:    true,
		}
		if env.KerberosKeytabPath != "" {
			kerberosConfig.AuthType = sarama.KRB5_KEYTAB_AUTH
			kerberosConfig.KeyTabPath = env.KerberosKeytabPath
		} else {
			kerberosConfig.AuthType = sarama.KRB5_USER_AUTH
		}

		config.Net.SASL.GSSAPI = kerberosConfig
	}

	err = config.Validate()
	if err != nil {
		logger.Panicw("Config not valid", err)
	}

	kc, err := sarama.NewConsumerGroup(
		env.BootstrapServers,
		env.GroupID, config)
	if err != nil {
		logger.Panicw("Error creating Kafka Consumer", err)
	}

	return &kafkasourceAdapter{
		kafkaClient: kc,
		topics:      env.Topics,

		ceClient: ceClient,
		logger:   logger,
		mt:       mt,
	}
}

func (a *kafkasourceAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Kafka Source Adapter")

	consumerGroup := consumerGroupHandler{
		adapter: a,
	}

	for {
		// `Consume` should be called inside an infinite loop, when a
		// server-side rebalance happens, the consumer session will need to be
		// recreated to get the new claims.
		if err := a.kafkaClient.Consume(ctx, a.topics, consumerGroup); err != nil {
			return err
		}
		if ctx.Err() != nil {
			return nil
		}
	}
}

func newTLSCertificatesConfig(tlsConfig *tls.Config, clientCert, clientKey string) (*tls.Config, error) {
	if clientCert != "" && clientKey != "" {
		cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
		if err != nil {
			return tlsConfig, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

func newTLSRootCAConfig(tlsConfig *tls.Config, caCertFile string) *tls.Config {
	if caCertFile != "" {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(caCertFile))
		tlsConfig.RootCAs = caCertPool
	}
	return tlsConfig
}
