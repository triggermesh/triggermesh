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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"net/http"

	v1alpha1 "github.com/triggermesh/triggermesh/pkg/apis/sources/v1alpha1"
	"github.com/triggermesh/triggermesh/pkg/client/generated/clientset/internalclientset/scheme"
	rest "k8s.io/client-go/rest"
)

type SourcesV1alpha1Interface interface {
	RESTClient() rest.Interface
	AWSCloudWatchLogsSourcesGetter
	AWSCloudWatchSourcesGetter
	AWSCodeCommitSourcesGetter
	AWSCognitoIdentitySourcesGetter
	AWSCognitoUserPoolSourcesGetter
	AWSDynamoDBSourcesGetter
	AWSEventBridgeSourcesGetter
	AWSKinesisSourcesGetter
	AWSPerformanceInsightsSourcesGetter
	AWSS3SourcesGetter
	AWSSNSSourcesGetter
	AWSSQSSourcesGetter
	AzureActivityLogsSourcesGetter
	AzureBlobStorageSourcesGetter
	AzureEventGridSourcesGetter
	AzureEventHubsSourcesGetter
	AzureIOTHubSourcesGetter
	AzureQueueStorageSourcesGetter
	AzureServiceBusQueueSourcesGetter
	AzureServiceBusSourcesGetter
	AzureServiceBusTopicSourcesGetter
	CloudEventsSourcesGetter
	GoogleCloudAuditLogsSourcesGetter
	GoogleCloudBillingSourcesGetter
	GoogleCloudPubSubSourcesGetter
	GoogleCloudSourceRepositoriesSourcesGetter
	GoogleCloudStorageSourcesGetter
	HTTPPollerSourcesGetter
	IBMMQSourcesGetter
	KafkaSourcesGetter
	OCIMetricsSourcesGetter
	SalesforceSourcesGetter
	SlackSourcesGetter
	SolaceSourcesGetter
	TwilioSourcesGetter
	WebhookSourcesGetter
	ZendeskSourcesGetter
}

// SourcesV1alpha1Client is used to interact with features provided by the sources.triggermesh.io group.
type SourcesV1alpha1Client struct {
	restClient rest.Interface
}

func (c *SourcesV1alpha1Client) AWSCloudWatchLogsSources(namespace string) AWSCloudWatchLogsSourceInterface {
	return newAWSCloudWatchLogsSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AWSCloudWatchSources(namespace string) AWSCloudWatchSourceInterface {
	return newAWSCloudWatchSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AWSCodeCommitSources(namespace string) AWSCodeCommitSourceInterface {
	return newAWSCodeCommitSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AWSCognitoIdentitySources(namespace string) AWSCognitoIdentitySourceInterface {
	return newAWSCognitoIdentitySources(c, namespace)
}

func (c *SourcesV1alpha1Client) AWSCognitoUserPoolSources(namespace string) AWSCognitoUserPoolSourceInterface {
	return newAWSCognitoUserPoolSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AWSDynamoDBSources(namespace string) AWSDynamoDBSourceInterface {
	return newAWSDynamoDBSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AWSEventBridgeSources(namespace string) AWSEventBridgeSourceInterface {
	return newAWSEventBridgeSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AWSKinesisSources(namespace string) AWSKinesisSourceInterface {
	return newAWSKinesisSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AWSPerformanceInsightsSources(namespace string) AWSPerformanceInsightsSourceInterface {
	return newAWSPerformanceInsightsSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AWSS3Sources(namespace string) AWSS3SourceInterface {
	return newAWSS3Sources(c, namespace)
}

func (c *SourcesV1alpha1Client) AWSSNSSources(namespace string) AWSSNSSourceInterface {
	return newAWSSNSSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AWSSQSSources(namespace string) AWSSQSSourceInterface {
	return newAWSSQSSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AzureActivityLogsSources(namespace string) AzureActivityLogsSourceInterface {
	return newAzureActivityLogsSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AzureBlobStorageSources(namespace string) AzureBlobStorageSourceInterface {
	return newAzureBlobStorageSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AzureEventGridSources(namespace string) AzureEventGridSourceInterface {
	return newAzureEventGridSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AzureEventHubsSources(namespace string) AzureEventHubsSourceInterface {
	return newAzureEventHubsSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AzureIOTHubSources(namespace string) AzureIOTHubSourceInterface {
	return newAzureIOTHubSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AzureQueueStorageSources(namespace string) AzureQueueStorageSourceInterface {
	return newAzureQueueStorageSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AzureServiceBusQueueSources(namespace string) AzureServiceBusQueueSourceInterface {
	return newAzureServiceBusQueueSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AzureServiceBusSources(namespace string) AzureServiceBusSourceInterface {
	return newAzureServiceBusSources(c, namespace)
}

func (c *SourcesV1alpha1Client) AzureServiceBusTopicSources(namespace string) AzureServiceBusTopicSourceInterface {
	return newAzureServiceBusTopicSources(c, namespace)
}

func (c *SourcesV1alpha1Client) CloudEventsSources(namespace string) CloudEventsSourceInterface {
	return newCloudEventsSources(c, namespace)
}

func (c *SourcesV1alpha1Client) GoogleCloudAuditLogsSources(namespace string) GoogleCloudAuditLogsSourceInterface {
	return newGoogleCloudAuditLogsSources(c, namespace)
}

func (c *SourcesV1alpha1Client) GoogleCloudBillingSources(namespace string) GoogleCloudBillingSourceInterface {
	return newGoogleCloudBillingSources(c, namespace)
}

func (c *SourcesV1alpha1Client) GoogleCloudPubSubSources(namespace string) GoogleCloudPubSubSourceInterface {
	return newGoogleCloudPubSubSources(c, namespace)
}

func (c *SourcesV1alpha1Client) GoogleCloudSourceRepositoriesSources(namespace string) GoogleCloudSourceRepositoriesSourceInterface {
	return newGoogleCloudSourceRepositoriesSources(c, namespace)
}

func (c *SourcesV1alpha1Client) GoogleCloudStorageSources(namespace string) GoogleCloudStorageSourceInterface {
	return newGoogleCloudStorageSources(c, namespace)
}

func (c *SourcesV1alpha1Client) HTTPPollerSources(namespace string) HTTPPollerSourceInterface {
	return newHTTPPollerSources(c, namespace)
}

func (c *SourcesV1alpha1Client) IBMMQSources(namespace string) IBMMQSourceInterface {
	return newIBMMQSources(c, namespace)
}

func (c *SourcesV1alpha1Client) KafkaSources(namespace string) KafkaSourceInterface {
	return newKafkaSources(c, namespace)
}

func (c *SourcesV1alpha1Client) OCIMetricsSources(namespace string) OCIMetricsSourceInterface {
	return newOCIMetricsSources(c, namespace)
}

func (c *SourcesV1alpha1Client) SalesforceSources(namespace string) SalesforceSourceInterface {
	return newSalesforceSources(c, namespace)
}

func (c *SourcesV1alpha1Client) SlackSources(namespace string) SlackSourceInterface {
	return newSlackSources(c, namespace)
}

func (c *SourcesV1alpha1Client) SolaceSources(namespace string) SolaceSourceInterface {
	return newSolaceSources(c, namespace)
}

func (c *SourcesV1alpha1Client) TwilioSources(namespace string) TwilioSourceInterface {
	return newTwilioSources(c, namespace)
}

func (c *SourcesV1alpha1Client) WebhookSources(namespace string) WebhookSourceInterface {
	return newWebhookSources(c, namespace)
}

func (c *SourcesV1alpha1Client) ZendeskSources(namespace string) ZendeskSourceInterface {
	return newZendeskSources(c, namespace)
}

// NewForConfig creates a new SourcesV1alpha1Client for the given config.
// NewForConfig is equivalent to NewForConfigAndClient(c, httpClient),
// where httpClient was generated with rest.HTTPClientFor(c).
func NewForConfig(c *rest.Config) (*SourcesV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	httpClient, err := rest.HTTPClientFor(&config)
	if err != nil {
		return nil, err
	}
	return NewForConfigAndClient(&config, httpClient)
}

// NewForConfigAndClient creates a new SourcesV1alpha1Client for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
func NewForConfigAndClient(c *rest.Config, h *http.Client) (*SourcesV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientForConfigAndClient(&config, h)
	if err != nil {
		return nil, err
	}
	return &SourcesV1alpha1Client{client}, nil
}

// NewForConfigOrDie creates a new SourcesV1alpha1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *SourcesV1alpha1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new SourcesV1alpha1Client for the given RESTClient.
func New(c rest.Interface) *SourcesV1alpha1Client {
	return &SourcesV1alpha1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1alpha1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *SourcesV1alpha1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
