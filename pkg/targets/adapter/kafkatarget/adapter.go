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

package kafkatarget

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"

	"github.com/Shopify/sarama"

	"github.com/triggermesh/triggermesh/pkg/apis/targets"
	"github.com/triggermesh/triggermesh/pkg/common/kafka"
	"github.com/triggermesh/triggermesh/pkg/metrics"
)

// NewTarget adapter implementation
func NewTarget(ctx context.Context, envAcc pkgadapter.EnvConfigAccessor, ceClient cloudevents.Client) pkgadapter.Adapter {
	logger := logging.FromContext(ctx)

	mt := &pkgadapter.MetricTag{
		ResourceGroup: targets.KafkaTargetResource.String(),
		Namespace:     envAcc.GetNamespace(),
		Name:          envAcc.GetName(),
	}

	metrics.MustRegisterEventProcessingStatsView()

	env := envAcc.(*envAccessor)

	var err error

	config := sarama.NewConfig()

	if env.SASLEnable {
		mechanism := sarama.SASLMechanism(env.SecurityMechanisms)

		// If the SASL SCRAM mechanism a SCRAM generator must be provided pointing
		// to a corresponding hash generator function.
		switch mechanism {
		case sarama.SASLTypeSCRAMSHA256:
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: sha256.New} }
		case sarama.SASLTypeSCRAMSHA512:
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: sha512.New} }
		}

		config.Net.SASL.Enable = env.SASLEnable
		config.Net.SASL.Mechanism = mechanism
		config.Net.SASL.User = env.Username
		config.Net.SASL.Password = env.Password
	}

	if env.TLSEnable {
		config.Net.TLS.Enable = env.TLSEnable

		tlsCfg := &tls.Config{}
		if env.CA != "" {
			addCAConfig(tlsCfg, env.CA)
		}

		if env.ClientCert != "" || env.ClientKey != "" {

			if err := addTLSCerts(tlsCfg, env.ClientCert, env.ClientKey); err != nil {
				logger.Panicw("Could not parse the TLS Certificates", zap.Error(err))
			}
		}

		config.Net.TLS.Config = tlsCfg
		config.Net.TLS.Config.InsecureSkipVerify = env.SkipVerify
	}

	if env.SecurityMechanisms == "GSSAPI" {
		kerberosConfig := sarama.GSSAPIConfig{
			AuthType:           sarama.KRB5_USER_AUTH,
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
		}

		config.Net.SASL.GSSAPI = kerberosConfig
	}

	config.Producer.Return.Successes = true

	scc, err := kafka.NewSaramaCachedClient(ctx, env.BootstrapServers, config,
		logger.Named("sarama").Desugar(),
		kafka.WithSaramaCachedClientRefresh(env.ConnectionRefreshPeriod),
	)
	if err != nil {
		logger.Panicw("Error creating kafka client", zap.Error(err))
	}

	return &kafkaAdapter{
		saramaCachedClient:        scc,
		topic:                     env.Topic,
		createTopicIfMissing:      env.CreateTopicIfMissing,
		flushTimeout:              env.FlushOnExitTimeoutMillisecs,
		topicTimeout:              env.CreateTopicTimeoutMillisecs,
		newTopicReplicationFactor: env.NewTopicReplicationFactor,
		newTopicPartitions:        env.NewTopicPartitions,

		discardCEContext: env.DiscardCEContext,

		ceClient: ceClient,
		logger:   logger,

		sr: metrics.MustNewEventProcessingStatsReporter(mt),
	}
}

var _ pkgadapter.Adapter = (*kafkaAdapter)(nil)

type kafkaAdapter struct {
	saramaCachedClient *kafka.SaramaCachedClient
	topic              string

	createTopicIfMissing bool

	flushTimeout              int
	topicTimeout              int
	newTopicReplicationFactor int16
	newTopicPartitions        int32

	discardCEContext bool

	ceClient cloudevents.Client
	logger   *zap.SugaredLogger

	sr *metrics.EventProcessingStatsReporter
}

func (a *kafkaAdapter) Start(ctx context.Context) error {
	a.logger.Info("Starting Kafka adapter")

	if err := a.saramaCachedClient.EnsureTopic(a.topic, a.newTopicReplicationFactor, a.newTopicPartitions); err != nil {
		return err
	}

	defer func() {
		if err := a.saramaCachedClient.Close(); err != nil {
			a.logger.Warnw("could not close kafka connections", zap.Error(err))
		}
	}()

	return a.ceClient.StartReceiver(ctx, a.dispatch)
}

func (a *kafkaAdapter) dispatch(event cloudevents.Event) cloudevents.Result {
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

	msg := &sarama.ProducerMessage{
		Topic: a.topic,
		Key:   sarama.StringEncoder(event.ID()),
		Value: sarama.ByteEncoder(msgVal),
	}

	if err := a.saramaCachedClient.SendMessageSync(msg); err != nil {
		a.logger.Errorw("Error producing Kafka message", zap.String("msg", string(msgVal)), zap.Error(err))
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
