/*
Copyright 2020-2021 TriggerMesh Inc.

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

	// HTTPPollerSourceResource represents an event source for polling HTTP endpoints.
	HTTPPollerSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "httppollersources",
	}

	// SlackSourceResource represents an event source for Slack.
	SlackSourceResource = schema.GroupResource{
		Group:    GroupName,
		Resource: "slacksources",
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
