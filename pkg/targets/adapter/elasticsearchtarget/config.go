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
	"crypto/tls"
	"crypto/x509"
	"net/http"

	"github.com/elastic/go-elasticsearch/v7"
	pkgadapter "knative.dev/eventing/pkg/adapter/v2"
)

// EnvAccessorCtor for configuration parameters
func EnvAccessorCtor() pkgadapter.EnvConfigAccessor {
	return &envAccessor{}
}

type envAccessor struct {
	pkgadapter.EnvConfig

	Addresses  []string `envconfig:"ELASTICSEARCH_ADDRESSES"`
	User       string   `envconfig:"ELASTICSEARCH_USER"`
	Password   string   `envconfig:"ELASTICSEARCH_PASSWORD"`
	APIKey     string   `envconfig:"ELASTICSEARCH_APIKEY"`
	CACert     string   `envconfig:"ELASTICSEARCH_CACERT"`
	SkipVerify bool     `envconfig:"ELASTICSEARCH_SKIPVERIFY" default:"false"`

	IndexName string `envconfig:"ELASTICSEARCH_INDEX" required:"true"`

	DiscardCEContext bool `envconfig:"ELASTICSEARCH_DISCARD_CE_CONTEXT"`

	// CloudEvents responses parametrization
	CloudEventPayloadPolicy string `envconfig:"EVENTS_PAYLOAD_POLICY" default:"always"`

	// BridgeIdentifier is the name of the bridge workflow this target is part of
	BridgeIdentifier string `envconfig:"EVENTS_BRIDGE_IDENTIFIER"`
}

func (e *envAccessor) GetElasticsearchConfig() *elasticsearch.Config {
	config := &elasticsearch.Config{
		Addresses: e.Addresses,
		APIKey:    e.APIKey,
		Username:  e.User,
		Password:  e.Password,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: e.SkipVerify,
				RootCAs:            createCACertPool(e.CACert),
			},
		},
		RetryOnStatus: []int{502, 503, 504, 429},
		MaxRetries:    10,
	}
	return config
}

func createCACertPool(cacert string) *x509.CertPool {
	cacerts, _ := x509.SystemCertPool()
	if cacerts == nil {
		cacerts = x509.NewCertPool()
	}
	if cacert != "" {
		cacerts.AppendCertsFromPEM([]byte(cacert))
	}
	return cacerts
}
