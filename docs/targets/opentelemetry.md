# OpenTelemetry Target

The OpenTelemetry target exposes a common interface to a range of metrics backends.

## Contents

- [OpenTelemetry Target](#opentelemetry-target)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Running Locally From Code](#running-locally-from-code)
  - [Running From Kubernetes](#running-from-kubernetes)
  - [Accepted CloudEvents](#accepted-cloudevents)
  - [Examples](#examples)

## Prerequisites

We will use Cortex to configure the OpenTelemetry adapter.

Setup and run [Cortex](https://github.com/cortexproject/cortex/), for testing purposes set the storage configuration option to `filesystem` and create the rules folder manually.

By default Cortex exposes its API at http://localhost:9009. These are some of the handy endpoints to use when testing this target:

- List all labels http://localhost:9009/api/prom/api/v1/labels
- Query metrics (PromQL) for instrument http://localhost:9009/api/prom/api/v1/query?query=INSTRUMENT-NAME

## Running Locally From Code

The  `./config` directory can be used to deploy the controller and all adapters. Please consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to a cluster.

The adapter code can be run directly from the top-level directory:

```sh
NAMESPACE=default \
K_METRICS_CONFIG={} \
K_LOGGING_CONFIG={} \
OPENTELEMETRY_CORTEX_ENDPOINT=http://localhost:9009/api/prom/push \
OPENTELEMETRY_INSTRUMENTS='[
	{"name":"total_requests","instrument":"Counter","number":"Int64","description":"total requests"},
  {"name":"quacking_ducks","instrument":"UpDownCounter","number":"Int64","description":"number of quacking ducks observed"},
  {"name":"request_duration_ms","instrument":"Histogram","number":"Float64","description":"request duration in milliseconds"},
	{"name":"cpu_usage","instrument":"UpDownCounter","number":"Float64","description":"cpu usage"}]' \
go run ./cmd/opentelemetrytarget-adapter/main.go
```

Note that several environment variables will need to be set prior to invoking the adapter such as:

  - `NAMESPACE=default`             - To match the one where this adapter is running.
  - `K_LOGGING_CONFIG=''`           - Define the default logging configuration.
  - `K_METRICS_CONFIG='''`          - Define the metrics configuration.
  - `OPENTELEMETRY_INSTRUMENTS`     - OpenTelemetry configured instruments.
  - `OPENTELEMETRY_CORTEX_ENDPOINT` - Cortex endpoint, only when using Cortex .

## Running From Kubernetes

TODO

## Accepted CloudEvents

TODO

## Examples

When running locally you can use `curl` and the CloudEvent examples in this section to test the adapter.

```console
curl -v -X POST http://localhost:8080  \
-H "content-type: application/json"  \
-H "ce-specversion: 1.0"  \
-H "ce-source: curl.client"  \
-H "ce-type: curl.sent.metrics"  \
-H "ce-id: 123-abc" \
-H "ce-statefulid: my-stateful-12345" \
-d '{
      "name":"total_requests",
      "kind":"Counter",
      "value":1,
      "attributes":[
        {"key":"host","value":"tm1","type":"string"},
        {"key":"type","value":"large","type":"string"}
      ]
    }'
```

This example uses a `Counter` which value is meant to be added to the existing. Counters are non decreasing monotonics instruments.

```json
{
      "name":"total_requests",
      "kind":"Counter",
      "value":1,
      "attributes":[
        {"key":"host","value":"tm1","type":"string"},
        {"key":"type","value":"large","type":"string"}
      ]
    }
```

This example uses an `UpDownCounter` which value is meant to be added to the existing. Counters can be increased and decreased by providing positive and nevative values respectively.

```json
{
      "name":"quacking_ducks",
      "kind":"UpDownCounter",
      "value":2,
      "attributes":[
        {"key":"is_mulard","value":false,"type":"bool"}
      ]
    }
```

An `Histogram` provides data to be bundled into populations and aggregated.

```json
{
      "name":"request_duration_ms",
      "kind":"Histogram",
      "value":52.1,
      "attributes":[
        {"key":"resource","value":"/job/process","type":"string"}
      ]
    }
```
