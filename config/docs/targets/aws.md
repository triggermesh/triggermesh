# AWS Target for Knative Eventing

This event target allows for invoking several types of AWS services.  Currently,
this target supports:
  * Lambdas
  * SNS
  * SQS
  * Kinesis
  * S3

## Prerequisites

Utilizing any of the AWS services requires that they already exist, and the AWS
credentials in use will have access to invoke the underlying services.

### Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please
consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to
a cluster.

The adapter can be built and invoked directly.  From the top-level source directory:

```sh
make aws-target-adapater && ./_output/aws-target-adapter
```

Note that several environment variables will need to be set prior to invoking the adapter such as:

  - `NAMESPACE=default`     - Usually set by the kubernetes cluster
  - `K_LOGGING_CONFIG=''`   - Define the default logging configuration
  - `K_METRICS_CONFIG='''`  - Define the prometheus metrics configuration
  - `AWS_ACCESS_KEY_ID`     - AWS API Access Key
  - `AWS_SECRET_ACCESS_KEY` - AWS API Secret
  - `AWS_TARGET_TYPE`       - Type of target this adapter will act as. One of `lambda`, `sns`, `sqs`, `kinesis`
  - `AWS_TARGET_ARN`        - Target ARN for the service being invoked
  - `AWS_KINESIS_PARTITION` - Kinesis partition name (only required for a `kinesis` target)

## Adding the AWS Secrets

A set of AWS API keys will need to be created and added to the same namespace
hosting the target, and would resemble:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws
type: Opaque
stringData:
  AWS_ACCESS_KEY_ID: <REPLACE ME WITH A REAL KEY>
  AWS_SECRET_ACCESS_KEY: <REPLACE ME WITH A REAL SECRET>
```

## Creating an AWS Service Target

Once the AWS Target Controller has been deployed, a target for Lambdas, SNS, SQS, and s3
can be created by defining their respective object. Most will be similar to below
where the `arn`, `accessKeyID`, and `secretAccessKey` will be required while the kind will
be one of:

  * AWSLambdaTarget
  * AWSSNSTarget
  * AWSSQSTarget
  * AWSKinesisTarget
  * AWSS3Target

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: AWSLambdaTarget
metadata:
  name: triggermesh-aws-lambda
spec:
  arn: arn:aws:lambda:us-west-2:043455440429:function:snslistener
  awsApiKey:
  auth:
    credentials:
      accessKeyID:
        valueFromSecret:
          name: aws
          key: AWS_ACCESS_KEY_ID
      secretAccessKey:
        valueFromSecret:
          name: aws
          key: AWS_SECRET_ACCESS_KEY
```

For more details, consult the [target samples](samples/aws).

_NOTE: The `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` secrets must be installed
and accessible to the target service_.

_NOTE: For the S3 target, the `subject` attribute of the received CloudEvent is
used to indicate what bucket key should be used. By default, the bucket key will be set to **Ce-Type**/**Ce-Source**/**Ce-Time**. When the `type` attribute
of the received CloudEvent is `io.triggermesh.awss3.object.put`, only the
CloudEvent data (without context attributes) is stored in the destination S3
object, regardless of the value of the `discardCloudEventContext` spec attribute._

## AWS Target as an Event Sink

Lastly, a triggering mechanism needs to be added to listen for a Knative
event.

```yaml
apiVersion: eventing.knative.dev/v1beta1
kind: Trigger
metadata:
  name: aws-sample-lambda-trigger
spec:
  broker: default
  subscriber:
    ref:
      apiVersion: targets.triggermesh.io/v1alpha1
      kind: AWSTarget
      name: triggermesh-aws-lambda

```

For additional samples, take a look at the [samples](../samples/aws).

## Triggering an AWS Service via the Target

The AWS Target can be triggered as a normal web service using a
tool such as `curl` within the cluster.  A sample message would resemble the
following:

```console
curl -v http://triggermesh-aws-lambda.default.svc.cluster.local \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: dev.knative.source.aws" \
 -H "Ce-Source: awesome/instance" \
 -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
 -d '{"greeting":"Hi from TriggerMesh"}'
```


### Sending events to the DynamoDB Target

Events can overwrite the default table name set at the spec by providing a table name at the `Ce-Source` attribute. 

```console
curl -v http://awstarget-triggermesh-aws-dynamodb.d.svc.cluster.local \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: io.triggermesh.aws.dynamodb.item.put" \
 -H "Ce-Subject: <TABLE_NAME>" \
 -H "Ce-Source: awesome/instance" \
 -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
 -d '{"Message":"Hi from TriggerMesh"}'
```
