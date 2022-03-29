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
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/kelseyhightower/envconfig"
)

const (
	cacert = `-----BEGIN CERTIFICATE-----
	MIIDiTCCAnGgAwIBAgIUL8bOyGrZR07jqJHX/Hz865A4fGEwDQYJKoZIhvcNAQEL
	BQAwVDELMAkGA1UEBhMCSU4xEDAOBgNVBAgMB0luZmVybm8xEDAOBgNVBAcMB2R1
	bmdlb24xITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMDA0
	MjExMzE5NTRaFw0yMjAyMTYxMzE5NTRaMFQxCzAJBgNVBAYTAklOMRAwDgYDVQQI
	DAdJbmZlcm5vMRAwDgYDVQQHDAdkdW5nZW9uMSEwHwYDVQQKDBhJbnRlcm5ldCBX
	aWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCd
	E0SKl4GIysk+xsJuF+0WoaA1po8rs2vQrTNzZXCZKz3KIezJQ/aLIMXxrgr901x3
	uRhnNszzBt6WI3wnDkIr8LUdXIALKy4XdPVdyAm3GSyH3rtn+q2d/KnsfzXisKKr
	mU5P355EvcQdn6OJYHSxIJ6r019JDsQRrRHp1v0qMC5LcXJ6F664EZoe1i4Mo1gJ
	FfqCFYJzXg2+Vm1lLDor8PSPm9C1BPHzqcbXof1Z08EPJXaCGZjJYhmVKc+aIEBl
	hDpM/TUES0bpFgcyPjvzy2BNtnGgwty2O7xtkCfybpo/JN1gp3NlkjEyDRqrprd1
	JJkeI0qNTa94o2YXg32dAgMBAAGjUzBRMB0GA1UdDgQWBBQ91TaeGnfBUnFTrjBB
	n7kZo2ESizAfBgNVHSMEGDAWgBQ91TaeGnfBUnFTrjBBn7kZo2ESizAPBgNVHRMB
	Af8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQAGz8LFjuKGMS+1yn5rx8p1mNd8
	3TmOkS58Jv+JZY/SuAtz0DnB2PGdL0U6L8RM1D1OdmokwR9yj3dA3/TGSokEujLY
	h3+WnIEh2r+JKQDuPbwM/Ym8DaW0xAxv6niKjknH0otSty/3mISnRB05NrYM67A1
	KiNCVWFfvzm7CMwgvxSFGBgCz5iTBzd34QkG5TbKuWOJHIi2N++NYPSl9ttoxAyM
	FxOaWiA4DARX/jKMY+QVYo3Gg08K3ulsvqugkOXRHdTIE8L1s638+tvQAozARlpN
	4PHpk+V9b3YXBgxd5/i/icjXciy/icI25bVQT7iKfnBgF/IbGA0CJjWgYaMY
	-----END CERTIFICATE-----
	`
)

func TestEnvironment(t *testing.T) {
	testCases := []struct {
		testName       string
		env            map[string]string
		expectedConfig *elasticsearch.Config
	}{{
		testName: "Elasticsearch user and password",
		env: map[string]string{
			"NAMESPACE":              "namespace-es",
			"K_LOGGING_CONFIG":       "{}",
			"K_METRICS_CONFIG":       "{}",
			"ELASTICSEARCH_USER":     "triggermesh_user",
			"ELASTICSEARCH_PASSWORD": "triggermesh_password",
			"ELASTICSEARCH_INDEX":    "index1",
		},
		expectedConfig: createElasticsearchConfig("triggermesh_user", "triggermesh_password", ""),
	}, {
		testName: "Elasticsearch API key",
		env: map[string]string{
			"NAMESPACE":            "namespace-es",
			"K_LOGGING_CONFIG":     "{}",
			"K_METRICS_CONFIG":     "{}",
			"ELASTICSEARCH_APIKEY": "triggermesh-api-key",
			"ELASTICSEARCH_INDEX":  "index1",
		},
		expectedConfig: createElasticsearchConfig("", "", "triggermesh-api-key"),
	}, {
		testName: "Elasticsearch addresses",
		env: map[string]string{
			"NAMESPACE":               "namespace-es",
			"K_LOGGING_CONFIG":        "{}",
			"K_METRICS_CONFIG":        "{}",
			"ELASTICSEARCH_APIKEY":    "triggermesh-api-key",
			"ELASTICSEARCH_ADDRESSES": "address1,address2",
			"ELASTICSEARCH_INDEX":     "index1",
		},
		expectedConfig: createElasticsearchConfig("", "", "triggermesh-api-key", esWithAddresses("address1", "address2")),
	}, {
		testName: "Elasticsearch skip verify",
		env: map[string]string{
			"NAMESPACE":                "namespace-es",
			"K_LOGGING_CONFIG":         "{}",
			"K_METRICS_CONFIG":         "{}",
			"ELASTICSEARCH_APIKEY":     "triggermesh-api-key",
			"ELASTICSEARCH_SKIPVERIFY": "true",
			"ELASTICSEARCH_INDEX":      "index1",
		},
		expectedConfig: createElasticsearchConfig("", "", "triggermesh-api-key", esConfigWithSkipVerify(true)),
	}, {
		testName: "Elasticsearch CA certificate",
		env: map[string]string{
			"NAMESPACE":                "namespace-es",
			"K_LOGGING_CONFIG":         "{}",
			"K_METRICS_CONFIG":         "{}",
			"ELASTICSEARCH_APIKEY":     "triggermesh-api-key",
			"ELASTICSEARCH_SKIPVERIFY": "false",
			"ELASTICSEARCH_CACERT":     cacert,
			"ELASTICSEARCH_INDEX":      "index1",
		},
		expectedConfig: createElasticsearchConfig("", "", "triggermesh-api-key", esConfigWithSkipVerify(false), esConfigWithCACert(cacert)),
	}, {
		testName: "Elasticsearch Index and skip verify",
		env: map[string]string{
			"NAMESPACE":                "namespace-es",
			"K_LOGGING_CONFIG":         "{}",
			"K_METRICS_CONFIG":         "{}",
			"ELASTICSEARCH_APIKEY":     "triggermesh-api-key",
			"ELASTICSEARCH_SKIPVERIFY": "true",
			"ELASTICSEARCH_CACERT":     cacert,
			"ELASTICSEARCH_INDEX":      "index1",
		},
		expectedConfig: createElasticsearchConfig("", "", "triggermesh-api-key", esConfigWithSkipVerify(true), esConfigWithCACert(cacert)),
	}}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			env := EnvAccessorCtor().(*envAccessor)
			err := envconfig.Process("", env)

			if err != nil {
				t.Fatalf("Error parsing environment: %s", err)
			}
			config := env.GetElasticsearchConfig()
			if !elasticSearchConfigIsEqual(tc.expectedConfig, config) {
				t.Fatalf("Expected configuration was: %+v\nbut got: %+v", tc.expectedConfig, config)
			}
		})
	}

}

func elasticSearchConfigIsEqual(expected, got *elasticsearch.Config) bool {
	if expected.APIKey != got.APIKey ||
		expected.Username != got.Username ||
		expected.Password != got.Password ||
		!reflect.DeepEqual(expected.Addresses, got.Addresses) {
		return false
	}

	expSkipVerify := false
	if expected.Transport != nil {
		t := expected.Transport.(*http.Transport)
		if t.TLSClientConfig != nil && t.TLSClientConfig.InsecureSkipVerify {
			expSkipVerify = true
		}
	}

	gott := got.Transport.(*http.Transport)
	if gott.TLSClientConfig.InsecureSkipVerify != expSkipVerify {
		fmt.Printf("this is the problme man\n")
		// if expected.Transport == nil {
		// 	return false
		// }
		// t := expected.Transport.(*http.Transport)
		// if t.TLSClientConfig == nil ||
		// 	t.TLSClientConfig.InsecureSkipVerify == false {
		return false
		//		}
	}

	// TODO check CACert

	return true
}

type elasticSearchConfigOptionFunc func(*elasticsearch.Config) *elasticsearch.Config

func createElasticsearchConfig(username, password, apikey string, opFn ...elasticSearchConfigOptionFunc) *elasticsearch.Config {
	c := &elasticsearch.Config{
		Username: username,
		Password: password,
		APIKey:   apikey,
	}
	for _, f := range opFn {
		c = f(c)
	}
	return c
}

func esWithAddresses(addresses ...string) elasticSearchConfigOptionFunc {
	return func(config *elasticsearch.Config) *elasticsearch.Config {
		config.Addresses = addresses
		return config
	}
}

func esConfigWithSkipVerify(value bool) elasticSearchConfigOptionFunc {
	return func(config *elasticsearch.Config) *elasticsearch.Config {
		if config.Transport == nil {
			config.Transport = &http.Transport{}
		}
		t := config.Transport.(*http.Transport)
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{}
		}
		t.TLSClientConfig.InsecureSkipVerify = value
		return config
	}
}

func esConfigWithCACert(cacert string) elasticSearchConfigOptionFunc {
	return func(config *elasticsearch.Config) *elasticsearch.Config {
		if config.Transport == nil {
			config.Transport = &http.Transport{}
		}
		t := config.Transport.(*http.Transport)
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{}
		}
		t.TLSClientConfig.RootCAs = createCACertPool(cacert)
		return config
	}
}
