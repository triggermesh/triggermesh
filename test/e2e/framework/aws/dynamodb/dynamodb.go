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

// Package dynamodb contains helpers for AWS DynamoDB.
package dynamodb

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"

	"github.com/triggermesh/triggermesh/test/e2e/framework"
)

type TestItem struct {
	MyValue string
}

const attrName = "MyValue"

// CreateTable creates a table named after the given framework.Framework.
func CreateTable(dc dynamodbiface.DynamoDBAPI, f *framework.Framework) string /*arn*/ {
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String(attrName),
				AttributeType: aws.String(dynamodb.ScalarAttributeTypeS),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String(attrName),
				KeyType:       aws.String(dynamodb.KeyTypeHash),
			},
		},
		StreamSpecification: &dynamodb.StreamSpecification{
			StreamEnabled:  aws.Bool(true),
			StreamViewType: aws.String(dynamodb.StreamViewTypeNewAndOldImages),
		},
		BillingMode: aws.String(dynamodb.BillingModePayPerRequest),
		TableName:   &f.UniqueName,
	}

	if _, err := dc.CreateTable(input); err != nil {
		framework.FailfWithOffset(2, "Failed to create table %q: %s", *input.TableName, err)
	}

	waitUntilTableExists(dc, input.TableName)

	output, err := dc.DescribeTable(&dynamodb.DescribeTableInput{TableName: input.TableName})
	if err != nil {
		framework.FailfWithOffset(2, "Failed to describe table %q: %s", *input.TableName, err)
	}

	return *output.Table.TableArn
}

// PutItem inserts a new Item into the table of the given name.
func PutItem(dc dynamodbiface.DynamoDBAPI, tableName, value string) {
	item := TestItem{
		MyValue: value,
	}

	av, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		framework.FailfWithOffset(2, "Failed to marshal item to map: %s", err)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: &tableName,
	}

	if _, err = dc.PutItem(input); err != nil {
		framework.FailfWithOffset(2, "Failed to put item to table: %s", err)
	}
}

// DeleteTable deletes a table by name.
func DeleteTable(dc dynamodbiface.DynamoDBAPI, name string) {
	input := &dynamodb.DeleteTableInput{
		TableName: &name,
	}

	if _, err := dc.DeleteTable(input); err != nil {
		framework.FailfWithOffset(2, "Failed to delete table %q: %s", *input.TableName, err)
	}
}

// waitUntilTableExists uses the DynamoDB API operation
// DescribeTable to wait for a condition to be met before returning.
// If the condition is not met within the max attempt window, an error will
// be returned. Based on the kinesis.WaitUntilStreamExists API call
func waitUntilTableExists(dc dynamodbiface.DynamoDBAPI, name *string) {
	ctx := aws.BackgroundContext()
	w := request.Waiter{
		Name:        "WaitUntilTableExists",
		MaxAttempts: 18,
		Delay:       request.ConstantWaiterDelay(5 * time.Second),
		Acceptors: []request.WaiterAcceptor{
			{
				State:   request.SuccessWaiterState,
				Matcher: request.PathWaiterMatch, Argument: "Table.TableStatus",
				Expected: dynamodb.TableStatusActive,
			},
		},

		NewRequest: func(opts []request.Option) (*request.Request, error) {
			req, _ := dc.DescribeTableRequest(&dynamodb.DescribeTableInput{TableName: name})
			req.SetContext(ctx)
			return req, nil
		},
	}

	if err := w.WaitWithContext(ctx); err != nil {
		framework.FailfWithOffset(3, "Failed while waiting for table to become ACTIVE: %s", err)
	}
}
