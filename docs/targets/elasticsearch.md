# Elasticsearch event target for Knative Eventing

This event target integrates with Elasticsearch, using received Cloud Event messages to index documents.

## Contents

- [Elasticsearch event target for Knative Eventing](#elasticsearch-event-target-for-knative-eventing)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Deploying From Code](#deploying-from-code)
  - [Creating an Elasticsearch Target](#creating-an-elasticsearch-target)
    - [Status](#status)
    - [Elasticsearch Target as an event Sink](#elasticsearch-target-as-an-event-sink)
    - [Indexing with the Elasticsearch Target](#indexing-with-the-elasticsearch-target)

## Prerequisites

A Elasticsearch cluster and a set of credentials:

- Elastic Cloud on Kubernetes [ECK](https://github.com/elastic/cloud-on-k8s/) is the simplest way to get started, Elastic Cloud or any other Elasticsearch cluster. Version 7.x is preferred.
- User and password to the Elasticsearch cluster.
- An APIKey instead of User and password.
- CACertificate if using self-signed certificate and `SkipVerify` is not configured.

## Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please
consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to
a cluster.

The adapter can be built and invoked directly.  From the top-level source directory:

```sh
make elasticsearch-target-adapater && ./_output/elasticsearch-target-adapter
```

Note that several environment variables will need to be set prior to invoking the adapter such as:

  - `NAMESPACE=default`        - Usually set by the kubernetes cluster
  - `K_LOGGING_CONFIG=''`      - Define the default logging configuration
  - `K_METRICS_CONFIG='''`     - Define the prometheus metrics configuration
  - `ELASTICSEARCH_ADDRESSES`  - Address of the elasticsearch server
  - `ELASTICSEARCH_USER`       - Elasticsearch user to interact on behalf of
  - `ELASTICSEARCH_PASSWORD`   - Password for the Elasticsearch user
  - `ELASTICSEARCH_APIKEY`     - API Key to interact with the Elasticsearch API 
  - `ELASTICSEARCH_CACERT`     - CA Certificate for the SSL cert used by Elasticsearch
  - `ELASTICSEARCH_SKIPVERIFY` - Skip SSL cert verification
  - `ELASTICSEARCH_INDEX`      - Index to write the results to 

A full deployment example is located in the [samples](../samples/elasticsearch) directory

## Creating an Elasticsearch Target

Once the Elasticsearch Target Controller has been deployed along all other needed assets are present we can create integrations by adding ElasticsearchTargets objects.

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: ElasticsearchTarget
metadata:
  name: <TARGET-NAME>
spec:
  connection:
    addresses:
      - <ELASTICSEARCH-URL>
    skipVerify: <true|false>
    caCert: <ELASTICSEARCH-CA-CERTIFICATE>
    apiKey:
      secretKeyRef:
        name: <SECRET-CONTAINING-APIKEY>
        key: <SECRET-KEY-CONTAINING-APIKEY>
    username: <ELASTICSEARCH-USERNAME>
    password:
      secretKeyRef:
        name: <SECRET-CONTAINING-PASSWORD>
        key: <SECRET-KEY-CONTAINING-PASSWORD>
  indexName: <ELASTICSEARCH-INDEX>
```

Connection must include at least one address, including protocol scheme and port.

- example: `https://elasticsearch-server:9200`

The connection must be filled with one of:

- `username` and `password`
- `apiKey`

If the Elasticsearch cluster is being served using a self-signed certificate the CA can be added, or TLS verify can be skipped:

- `caCert` for adding the PEM string for the certificate.
- `skipVerify` set to true for skip checking certificates.

Received events will be indexed using `indexName` as the elasticsearch index.

### Status

ElasticsearchTarget requires one of `password` or `apiKey` Secrets to be provided, once they are a Knative service will be created. Logs at the controller and kubernetes events can provide detailed information about the target reconciliation process.
A Status summary is added to the ElasticsearchTarget object informing of the all conditions that the target needs.

When ready the `status.address.url` will point to the internal point where Cloud Events should be sent.

### Elasticsearch Target as an event Sink

Elasticsearch Target is addressable, which means you can use it as a Sink for Knative components.

```yaml
apiVersion: eventing.knative.dev/v1beta1
kind: Trigger
metadata:
  name: <TRIGGER-NAME>
spec:
  broker: <BROKER-NAME>
  filter:
    attributes:
      type: <MESSAGE-TYPES-ES-PAYLOAD-FORMATTED>
  subscriber:
    ref:
      apiVersion: targets.triggermesh.io/v1alpha1
      kind: ElasticsearchTarget
      name: <TARGET-NAME>
```

### Indexing with the Elasticsearch Target

Elasticsearch Target will forward any JSON payload at the CloudEvent to be indexed. The only requirement is that it is a valid JSON.

It would be desirable that the JSON conforms to the index mapping at elasticsearch.

You can use `curl` from a container in the cluster pointing to the Elasticsearch target exposed URL:

```console
curl -v http://elasticsearchtarget-es-indexinge5d0adf0209a48c23fa958aa1b8ecf0b.default.svc.cluster.local \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: something.to.index.type" \
 -H "Ce-Source: some.origin/intance" \
 -H "Ce-Id: 536808d3-88be-4077-9d7a-a3f162705f79" \
 -d '{"message":"thanks for indexing this message","from": "TriggerMesh targets", "some_number": 12}'
```
