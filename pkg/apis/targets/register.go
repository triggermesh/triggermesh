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

package targets

import "k8s.io/apimachinery/pkg/runtime/schema"

// GroupName is the name of the API group this package's resources belong to.
const GroupName = "targets.triggermesh.io"

var (
	// AWSComprehendTargetResource respresents an event target for AWS Comprehend.
	AWSComprehendTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awscomprehendtargets",
	}
	// AWSDynamodbTargetResource respresents an event target for AWS DynamoDB.
	AWSDynamoDBTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awsdynamodbtargets",
	}
	// AWSEventbridgeTargetResource respresents an event target for AWS Event Bridge.
	AWSEventBridgeTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awseventbridgetargets",
	}
	// AWSKinesisTargetResource respresents an event target for AWS Kinesis.
	AWSKinesisTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awskinesistargets",
	}
	// AWSLambdaTargetResource respresents an event target for AWS Lambda.
	AWSLambdaTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awslambdatargets",
	}
	// AWSS3TargetResource respresents an event target for AWS S3.
	AWSS3TargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awss3targets",
	}
	// AWSSNSTargetResource respresents an event target for AWS SNS.
	AWSSNSTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awssnstargets",
	}
	// AWSSQSTargetResource respresents an event target for AWS SQS.
	AWSSQSTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "awssqstargets",
	}
	// AzureEventHubsTargetResource respresents an event target for Azure EventHubs.
	AzureEventHubsTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "azureeventhubstargets",
	}
	// AzureSentinelTargetResource respresents an event target for Azure Sentinel.
	AzureSentinelTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "azuresentineltargets",
	}
	// AzureServiceBusTargetResource respresents an event target for Azure Service Bus.
	AzureServiceBusTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "azureservicebustargets",
	}
	// CloudEventsTargetResource respresents an event target for CloudEvents gateway.
	CloudEventsTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "cloudeventstargets",
	}
	// DatadogTargetResource respresents an event target for Datadog.
	DatadogTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "datadogtargets",
	}
	// ElasticsearchTargetResource respresents an event target for Elasticsearch.
	ElasticsearchTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "elasticsearchtargets",
	}
	// GoogleCloudFirestoreTargetResource respresents an event target for Google Firestore.
	GoogleCloudFirestoreTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "googlecloudfirestoretargets",
	}
	// GoogleCloudStorageTargetResource respresents an event target for Google Storage.
	GoogleCloudStorageTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "googlecloudstoragetargets",
	}
	// GoogleCloudWorkflowsTargetResource respresents an event target for Google Workflows.
	GoogleCloudWorkflowsTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "googlecloudworkflowstargets",
	}
	// GoogleCloudPubSubTargetResource respresents an event target for Google Pub/Sub.
	GoogleCloudPubSubTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "googlecloudpubsubtargets",
	}
	// GoogleSheetTargetResource respresents an event target for Google Sheet.
	GoogleSheetTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "googlesheettargets",
	}
	// HTTPTargetResource respresents an event target for HTTP endpoint.
	HTTPTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "httptargets",
	}
	// IBMMQTargetResource respresents an event target for IBM MQ.
	IBMMQTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "ibmmqtargets",
	}
	// JiraTargetResource respresents an event target for Jira.
	JiraTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "jiratargets",
	}
	// KafkaTargetResource respresents an event target for Kafka.
	KafkaTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "kafkatargets",
	}
	// LogzTargetResource respresents an event target for Logz.
	LogzTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "logztargets",
	}
	// LogzMetricsTargetResource respresents an event target for Logz Metrics.
	LogzMetricsTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "logzmetricstargets",
	}
	// MongoDBTargetResource respresents an event target for MongoDB.
	MongoDBTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "mongodbtargets",
	}
	// OpenTelemetryTargetResource respresents an event target for OpenTelemetry.
	OpenTelemetryTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "opentelemetrytargets",
	}
	// OracleTargetResource respresents an event target for Oracle.
	OracleTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "oracletargets",
	}
	// SalesforceTargetResource respresents an event target for Salesforce.
	SalesforceTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "salesforcetargets",
	}
	// SendgridTargetResource respresents an event target for Sendgrid.
	SendgridTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "sendgridtargets",
	}
	// SlackTargetResource respresents an event target for Slack.
	SlackTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "slacktargets",
	}
	// SolaceTargetResource respresents an event target for Solace.
	SolaceTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "solacetargets",
	}
	// SplunkTargetResource respresents an event target for Splunk.
	SplunkTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "splunktargets",
	}
	// TwilioTargetResource respresents an event target for Twilio.
	TwilioTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "twiliotargets",
	}
	// ZendeskTargetResource respresents an event target for Zendesk.
	ZendeskTargetResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "zendesktargets",
	}
)
