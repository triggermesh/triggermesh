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

package sources

import "k8s.io/apimachinery/pkg/runtime/schema"

// GroupName is the name of the API group this package's resources belong to.
const GroupName = "sources.triggermesh.io"

var (
	// AWSCloudWatchSourceResource respresents an event source for Amazon CloudWatch.
	AWSCloudWatchSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awscloudwatchsources",
	}

	// AWSCloudWatchLogsSourceResource respresents an event source for Amazon CloudWatch Logs.
	AWSCloudWatchLogsSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awscloudwatchlogssources",
	}

	// AWSCodeCommitSourceResource respresents an event source for AWS CodeCommit.
	AWSCodeCommitSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awscodecommitsources",
	}

	// AWSCognitoIdentitySourceResource respresents an event source for Amazon Cognito Identity.
	AWSCognitoIdentitySourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awscognitoidentitysources",
	}

	// AWSCognitoUserPoolSourceResource respresents an event source for Amazon Cognito User Pool.
	AWSCognitoUserPoolSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awscognitouserpoolsources",
	}

	// AWSDynamoDBSourceResource respresents an event source for Amazon DynamoDB.
	AWSDynamoDBSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awsdynamodbsources",
	}

	// AWSEventBridgeSourceResource respresents an event source for Amazon EventBridge.
	AWSEventBridgeSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awseventbridgesources",
	}

	// AWSKinesisSourceResource respresents an event source for Amazon Kinesis.
	AWSKinesisSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awskinesissources",
	}

	// AWSPerformanceInsightsSourceResource respresents an event source for Amazon Performance Insights.
	AWSPerformanceInsightsSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awsperformanceinsights",
	}

	// AWSS3SourceResource respresents an event source for Amazon S3.
	AWSS3SourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awss3sources",
	}

	// AWSSNSSourceResource respresents an event source for Amazon SNS.
	AWSSNSSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awssnssources",
	}

	// AWSSQSSourceResource respresents an event source for Amazon SQS.
	AWSSQSSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awssqssources",
	}

	// AzureActivityLogsSourceResource respresents an event source for Azure
	// activity logs (part of Azure Monitor).
	// https://docs.microsoft.com/en-us/azure/azure-monitor/platform/activity-log
	AzureActivityLogsSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "azureactivitylogssources",
	}

	// AzureBlobStorageSourceResource respresents an event source for Azure Blob
	// Storage containers.
	AzureBlobStorageSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "azureblobstoragesources",
	}

	// AzureEventGridSourceResource respresents an event source for Azure Event Grid.
	AzureEventGridSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "azureeventgridsources",
	}

	// AzureEventHubsSourceResource respresents an event source for Azure Event Hub.
	AzureEventHubsSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "azureeventhubssources",
	}

	// AzureIOTHubSourceResource respresents an event source for Azure IOT Hub.
	AzureIOTHubSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "azureiothubsources",
	}

	// AzureQueueStorageSourceResource respresents an event source for Azure Queue Storage.
	AzureQueueStorageSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "azurequeuestoragesources",
	}

	// AzureServiceBusQueueSourceResource respresents an event source for
	// Azure Service Bus Queues.
	AzureServiceBusQueueSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "azureservicebusqueuesources",
	}

	// AzureServiceBusSourceResource respresents an event source for
	// Azure Service Bus.
	AzureServiceBusSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "azureservicebussources",
	}

	// AzureServiceBusTopicSourceResource respresents an event source for
	// Azure Service Bus Topics.
	AzureServiceBusTopicSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "azureservicebustopicsources",
	}

	// CloudEventsSourceResource respresents an event source for CloudEvents.
	CloudEventsSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "cloudeventssources",
	}

	// KafkaSourceResource respresents an event source for Kafka.
	KafkaSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "kafkasources",
	}

	// GoogleCloudAuditLogsSourceResource respresents an event source for Google
	// Cloud Audit Logs.
	GoogleCloudAuditLogsSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "googlecloudauditlogssources",
	}

	// GoogleCloudBillingSourceResource respresents an event source for Google
	// Cloud Billing.
	GoogleCloudBillingSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "googlecloudbillingsources",
	}

	// GoogleCloudPubSubSourceResource respresents an event source for Google Cloud
	// Pub/Sub.
	GoogleCloudPubSubSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "googlecloudpubsubsources",
	}

	// GoogleCloudSourceRepositoriesSourceResource respresents an event source for Google
	// Cloud Source Repositories.
	GoogleCloudSourceRepositoriesSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "googlecloudsourcerepositoriessources",
	}

	// GoogleCloudStorageSourceResource respresents an event source for Google
	// Cloud Storage.
	GoogleCloudStorageSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "googlecloudstoragesources",
	}

	// HTTPPollerSourceResource represents an event source for polling HTTP endpoints.
	HTTPPollerSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "httppollersources",
	}

	// IBMMQSourceResource respresents an event source for IBM MQ.
	IBMMQSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "ibmmqsources",
	}

	// OCIMetricsSourceResource represents an event source for OCI Metrics.
	OCIMetricsSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "ocimetricssources",
	}

	// MongoDBSourceResource represents an event source for MongoDB.
	MongoDBSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "mongodbsources",
	}

	// SalesforceSourceResource represents an event source for Salesforce.
	SalesforceSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "salesforcesources",
	}

	// SlackSourceResource represents an event source for Slack.
	SlackSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "slacksources",
	}

	// SolaceSourceResource represents an event source for Solace.
	SolaceSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "solacesources",
	}

	// TwilioSourceResource represents an event source for Twilio.
	TwilioSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "twiliosources",
	}

	// WebhookSourceResource represents an event source for HTTP webhooks.
	WebhookSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "webhooksources",
	}

	// ZendeskSourceResource represents an event source for Zendesk.
	ZendeskSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "zendesksources",
	}
)
