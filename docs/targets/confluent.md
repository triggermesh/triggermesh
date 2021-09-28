# Confluent event target for Knative Eventing

This event target integrates with Confluent, using received Cloud Event messages to publish a message to a Confluent Kafka Cluster.

## Contents

- [Confluent event target for Knative Eventing](#confluent-event-target-for-knative-eventing)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Deploying from Code](#deploying-from-code)
  - [Creating a Confluent Target](#creating-a-confluent-target)
    - [Status](#status)
    - [Confluent Target as an event Sink](#confluent-target-as-an-event-sink)
    - [Sending messages to the Confluent Target](#sending-messages-to-the-confluent-target)

## Prerequisites

A Confluent API key is required to utilize this target. For more information on
how to obtain one, see the [Confluent Docs](https://docs.confluent.io/5.3.1/cloud/using/api-keys.html)

## Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please
consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to
a cluster.

The adapter can be built and invoked directly.  From the source directory:

```sh
make confluent-target-adapater && ./_output/confluent-target-adapter
```

Note that several environment variables will need to be set prior to invoking
the adapter in addition to the core set specified in the [development guide](../DEVELOPMENT.md).
The Confluent specific environment variables can be found in the
[adapter source code](../pkg/adapter/confluenttarget/config.go).

## Creating a Confluent Target

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: ConfluentTarget
metadata:
  name: triggermesh-confluent
spec:
  saslMechanism: PLAIN
  securityProtocol: SASL_SSL

  # The following settings are REQUIRED.
  topic: test1
  bootstrapservers: <Bootstrap Server>
  username:
    secretKeyRef:
      name: confluent
      key: username
  password:
    secretKeyRef:
      name: confluent
      key: password

  # When a topic is created using this target these parameters should be used
  topicReplicationFactor: 3
  topicPartitions: 1
```

The following *ARE NOT* optional and without them the adapter will not deploy:

* topic
* boostrapservers
* saslMechanism
* securityProtocol
* username
* password

A full deployment example is located in the [samples](../samples/confluent) directory

### Status

ConfluentTarget requires two secrets to be provided for `username` and
`password`. Once they are present it will create a Knative Service. Controller
logs and events can provide detailed information about the process. A Status
summary is added to the ConfluentTarget object informing of the all conditions
that the target needs.

When ready the `status.address.url` will point to the internal point where Cloud Events should be sent.

### Confluent Target as an event Sink

A Confluent Target is addressable, which means it can be used as a Sink for Knative components.

```yaml
apiVersion: eventing.knative.dev/v1beta1
kind: Trigger
metadata:
  name: confluent-sample-trigger
spec:
  broker: default
  subscriber:
    ref:
      apiVersion: targets.triggermesh.io/v1alpha1
      kind: ConfluentTarget
      name: triggermesh-confluent
```

* A sample sink binding to a Confluent Target deployment.

```yaml
apiVersion: sources.triggermesh.io/v1alpha1
kind: <Sample Source>
metadata:
  name: <Sample Source Name>
spec:
  sampleToken:
    secretKeyRef:
      name: <sample>
      key: <sample key>
  sink:
    ref:
      apiVersion: targets.triggermesh.io/v1alpha1
      kind: ConfluentTarget
      name: triggermesh-confluent
```

### Sending messages to the Confluent Target

 A Confluent Target will, by default, accept any CloudEvent and pass the entire event into a message body.

 If the JSON payload of the CloudEvent includes a `message` parameter, only the string found in `message` will be posted in a message body.


`Curl` can be used from a container in the cluster pointing to the ConfluentTarget exposed URL:

```console
curl -v http://confluenttarget-int1-8dc3abc7d44bdd0130bd0a311bea272f.knative-samples.svc.cluster.local
 \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: some.message.type" \
 -H "Ce-Source: some.origin/intance" \
 -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
 -d '{"message":"Hello from TriggerMesh using Confluent!"}'
```
