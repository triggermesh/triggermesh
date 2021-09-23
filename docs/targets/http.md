# HTTP Event Target for Knative Eventing

This event target uses CloudEvents to consume a generic HTTP API.

## Contents

- [HTTP Event Target for Knative Eventing](#http-event-target-for-knative-eventing)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Deploying from Code](#deploying-from-code)
  - [Create HTTP Target Integration](#create-http-target-integration)
    - [Creating a HTTP Target](#creating-a-http-target)
  - [Using the HTTP Target](#using-the-http-target)
    - [COVID-19 stats](#covid-19-stats)
    - [Calendarific country calendar](#calendarific-country-calendar)

## Prerequisites

The HTTP event target sends requests to arbitrary URLs and wraps responses in CloudEvents back to the caller. Any HTTP endpoint that can be reached using basic authentication, any sort of static token or no authentication can be configured using this target.

## Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please
consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to
a cluster.

The adapter can be built and invoked directly. From the top-level source directory:

```sh
make http-target-adapater && ./_output/http-target-adapter
```

Several environment variables will need to be set prior to invoking the adapter such as:

- `NAMESPACE` at Kubernetes where the adapter is being run. Mandatory.
- `K_LOGGING_CONFIG` logging configuration as defined at Knative. Mandatory
- `K_METRICS_CONFIG` metrics configuration as defined at Knative. Mandatory
- `HTTP_EVENT_TYPE` event type for the response message. Mandatory
- `HTTP_EVENT_SOURCE` event source for the response message. Mandatory
- `HTTP_URL` including path and querystring. Mandatory
- `HTTP_METHOD` verb for the HTTP rquest. Mandatory
- `HTTP_SKIP_VERIFY` to skip remote server TLS certificate verification. Optional
- `HTTP_CA_CERTIFICATE` CA certificate configured for TLS connection. Optional
- `HTTP_BASICAUTH_USERNAME` basic authentication user name. Optional
- `HTTP_BASICAUTH_PASSWORD` basic authentication password. Optional
- `HTTP_HEADERS` extra headers that will be set on requests. Optional
- `HTTP_OAUTH_CLIENT_ID` OAuth client ID. Optional
- `HTTP_OAUTH_CLIENT_SECRET` OAuth client secret. Optional
- `HTTP_OAUTH_TOKEN_URL` authentication token URL. Optional
- `HTTP_OAUTH_SCOPE` comma separated list of scopes. Optional

## Create HTTP Target Integration

Configuring the HTTP integration might require some preparation on the remote service to integrate regarding not only URL but query strings, connection security and authentication.

All fixed items for the target should be part of its definition while parametrized values should be part of each request to this target.

When using basic authentication the password needs to be referenced through a Kubernetes secret.

### Creating a HTTP Target

The HTTP Target is a service which is able to receive CloudEvents and
transform them into method calls to an external HTTP API:

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: HTTPTarget
metadata:
  name: triggermesh-http
  namespace: mynamespace
spec:
  response:
    eventType: triggermesh.http.type
    eventSource: my.service.com
  endpoint: 'https://my.service.com/my/path?some_key=<SOME-KEY>'
  method: 'GET'
  skipVerify: false
  caCertificate: |-
    -----BEGIN CERTIFICATE-----
    MIIFazCCA1OgAwIBAgIUc6d3XTcIV4Ku7lovbHGuaVwAPqEwDQYJKoZIhvcNAQEL
    BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
    ...
    L4uCwbnED802y7PXCqNzcDjbRfWcXm2aDVM6Dc++am5NDx+JjTLFgNeiiAyRGI8z
    5tJeGYFpd4Cxzt92s6ODIZVZZe+vP41Jey23yEgPpyv5F47WegApe73g1y4bBjg=
    -----END CERTIFICATE-----
  basicAuthUsername: myuser
  basicAuthPassword:
    secretKeyRef:
      name: myservice
      key: password
  headers:
    User-Agent: Triggermesh-HTTP
    Some-header: some-value
```

Fields at the `spec` above e match those needed for the adapter:

- `response.eventType` event type for the response message. Mandatory
- `response.eventSource` event source for the response message. Mandatory
- `endpoint` URL including path and querystring for the remote HTTP service. Mandatory
- `method` verb for the HTTP rquest. Mandatory
- `skipVerify` to skip remote server TLS certificate verification. Optional
- `caCertificate` CA certificate configured for TLS connection. Optional
- `basicAuthUsername` basic authentication user name. Optional
- `basicAuthPassword` secret reference to basic authentication password. Optional
- `headers` string map of key/value pairs as HTTP headers. Optional

Once created the HTTP Target service will be ready to consume incoming CloudEvents.

## Using the HTTP Target

CloudEvents consumed by this target should include a valid JSON message
containing these optional fields.

```json
{
  "query_string": "var1=value1&var2=value2",
  "path_suffix": "order/30/item/10",
  "body": "{\"hello\":\"world\"}"
}
```

- `query_string` will be added to the target configured query string.
- `path_suffix` will be added to the target configured path.
- `body` will be set as the request's body.

### COVID-19 stats

We will configure an HTTP target that can use the [COVID-19 API](https://covid19api.com/). Then we will use it to gather information about the world total stats.

Create the HTTP Target pointing to the COVID-19 API:

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: HTTPTarget
metadata:
  name: corona
  namespace: mynamespace
spec:
  response:
    eventType: covid.stats
  endpoint: 'https://api.covid19api.com/'
  method: 'GET'
```

The target will expose an internal URL that can be retrieved using the Kubernetes API.

```sh
$ kubectl get httptargets.targets.triggermesh.io -n mynamespace
NAME                        URL                                                                     READY   REASON   AGE
corona   http://httptarget-corona-mynamespace.default.svc.cluster.local   True             5d5h
```

Run an ephemeral curl container passing the command CloudEvent parameters that will be adding the path suffix to the endpoint that returns the world total stats for the service.

```sh
$ kubectl run --generator=run-pod/v1 curl-me --image=curlimages/curl -ti --rm -- \
  -v -X POST http://httptarget-corona.mynamespace.svc.cluster.local \
  -H "content-type: application/json" \
  -H "ce-specversion: 1.0" \
  -H "ce-source: curl-triggermesh" \
  -H "ce-type: my-curl-type" \
  -H "ce-id: 123-abc" \
  -d '{"path_suffix":"world/total"}'

...

```

### Calendarific country calendar

We will configure an HTTP target that uses [Calendarify](https://calendarific.com/) to retrieve wordlwide holidays.

Create a [Calendarific account](https://calendarific.com/signup) and retrieve an API key.

Create the HTTP Target using the API key:

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: HTTPTarget
metadata:
 name: calendarific
 namespace: mynamespace
spec:
  response:
    eventType: calendarific.holidays
  endpoint: 'https://calendarific.com/api/v2/holidays?api_key=REPLACE-WITH-APIKEY'
  method: 'GET'
```

Retrieve the internal URL.

```sh
$ kubectl get httptargets.targets.triggermesh.io -n mynamespace
NAME                        URL                                                                     READY   REASON   AGE
calendarific  http://httptarget-calendarific-mynamespace.default.svc.cluster.local   True             5d5h
```

Run an ephemeral curl container passing the command CloudEvent parameters that will be adding the querystring to return the US holidays for 2021.

```sh
$ kubectl run --generator=run-pod/v1 curl-me --image=curlimages/curl -ti --rm -- \
  -v -X POST http://httptarget-calendarific.mynamespace.svc.cluster.local \
  -H "content-type: application/json" \
  -H "ce-specversion: 1.0" \
  -H "ce-source: curl-triggermesh" \
  -H "ce-type: my-curl-type" \
  -H "ce-id: 123-abc" \
  -d '{"query_string":"country=US&year=2020"}'

...

```
