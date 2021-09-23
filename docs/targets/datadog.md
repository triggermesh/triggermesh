# Datadog Event Target for Knative Eventing

This event target integrates with Datadog, using received CloudEvent messages to
push metric or event data to Datadog. Enabling the end user to drive different forms of data-visualization via event-driven data. 

## Contents
- [Prerequisites](#Prerequisites)
- [Deploying from Code](#deploying-from-code])
- [Creating a Datadog Target](#creating-a-datadog-target)
- [Event Types](#event-types)
  - [io.triggermesh.datadog.metric](#io.triggermesh.datadog.metric)

## Prerequisites

A Datadog API key will be required to publish to Datadog.

## Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please
consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to
a cluster.

The adapter can be built and invoked directly.  From the top-level source directory:

```sh
make datadog-target-adapter && ./_output/datadog-target-adapter
```

A full deployment example is located in the [samples](../samples/datadog) directory

## Creating a Datadog Target

Once the Datadog Target Controller has been deployed, A Kubernetes secret in the same namespace must be created containing a key:`apiKey`. With a value populated by a valid Datadog API Key. 

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ddapitoken
type: Opaque
stringData:
  apiKey: __API_KEY__
```

A Datadog Target can be created by using the following YAML:

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: DatadogTarget
metadata:
 name: datadogtarget
spec:
 apiKey:
  secretKeyRef:
    name: ddapitoken
    key: apiKey
```

A valid `apiKey` is ***REQUIRED*** to deploy


## Event Types


### io.triggermesh.datadog.metric

Events of this type intend to post a metric to Datadog

#### Example CE posting an event of type "io.triggermesh.datadog.metric.submit"

```cmd
curl -v "http://localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.datadog.metric.submit" \
       -H "Ce-Source: ocimetrics/adapter" \
       -H "Content-Type: application/json" \
       -d '{"series":[{"metric":"five.golang","points":[["1614962026","14.5"]]}]}'
```


#### Example CE posting an event of type "io.triggermesh.datadog.event.post"

```cmd
curl -v "http://localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.datadog.event.post" \
       -H "Ce-Source: ocimetrics/adapter" \
       -H "Content-Type: application/json" \
       -d '{"text": "Oh boy2!","title": "Did you hear the news today?"}'
```

#### Example CE posting an event of type "io.triggermesh.datadog.logs.send"

```cmd
curl -v "http://localhost:8080" \
       -X POST \
       -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
       -H "Ce-Specversion: 1.0" \
       -H "Ce-Type: io.triggermesh.datadog.log.send" \
       -H "Ce-Source: ocimetrics/adapter" \
       -H "Content-Type: application/json" \
       -d '{  "ddsource": "nginx", "ddtags": "env:staging,version:5.1", "hostname": "i-012345678", "message": "2019-11-19T14:37:58,995 INFO Hello World", "service": "payment"}'
```
