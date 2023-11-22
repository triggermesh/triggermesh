# Salesforce Knative Target

This event target integrates with Salesforce, using received CloudEvent messages to perform request against the Salesforce API.

## Contents

- [Salesforce Knative Target](#salesforce-knative-target)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Deploying from Code](#deploying-from-code)
  - [Creating a Salesforce Token Secret](#creating-a-salesforce-token-secret)
  - [Creating a Salesforce Target](#creating-a-salesforce-target)
    - [Sending Messages to the Salesforce Target](#sending-messages-to-the-salesforce-target)
      - [Examples](#examples)

## Prerequisites

1. Salesforce instance.
1. Salesforce OAuth JWT credentials.

To create the credentials set follow the instructions at [the Salesforce documentation](https://help.salesforce.com/articleView?id=sf.remoteaccess_oauth_jwt_flow.htm)

## Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to a cluster.

The adapter can be built and invoked directly.  From the top-level source directory:

```sh
make salesforce-target-adapater && ./_output/salesforce-target-adapter
```

Note that several environment variables will need to be set prior to invoking the adapter such as:

- `NAMESPACE=default`          - Usually set by the kubernetes cluster.
- `K_LOGGING_CONFIG=''`        - Define the default logging configuration.
- `K_METRICS_CONFIG='''`       - Define the prometheus metrics configuration.
- `SALESFORCE_AUTH_CLIENT_ID`  - Salesforce OAuth Client ID.
- `SALESFORCE_AUTH_USER`       - Salesforce OAuth User.
- `SALESFORCE_AUTH_SERVER`     - Salesforce OAuth Server URL.
- `SALESFORCE_AUTH_CERT_KEY`   - Salesforce OAuth Certificate signing Key.
- `SALESFORCE_API_VERSION`     - Salesforce API Version (optional).

A full deployment example is located in the [samples](../samples/salesforce) directory

## Creating a Salesforce Token Secret

To access the Salesforce services OAuth JWT credentials are needed. The private key to sign certificates need to be created through a secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: salesforce
type: Opaque
stringData:
  certKey: |-
    -----BEGIN PRIVATE KEY-----
    ...certificate...contents...
    -----END PRIVATE KEY-----
```

## Creating a Salesforce Target

An example of a Salesforce target for a function would resemble the following:

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: SalesforceTarget
metadata:
  name: salesforce
spec:
  auth:
    clientID: my.salesforce.client.id
    server: https://login.salesforce.com
    user: woodford@triggermesh.com
    certKey:
      secretKeyRef:
        name: salesforce
        key: certKey
  apiVersion: v50.0
  eventOptions:
    payloadPolicy: always
```

- All fields in the `spec.auth` path are required.
- Field `apiVersion` is optional, when not informed the latest version will be used.
- Event options include the `payloadPolicy` which specifies if responses should be sent. Possible values are `always`, `error` and `never`. Default value is `always`.

### Sending Messages to the Salesforce Target

The Salesforce target accepts the event type `io.triggermesh.salesforce.apicall` and returns `io.triggermesh.salesforce.apicall.response`

The payload contains a JSON structure with elements to execute the API request:

- `action`: is the HTTP verb to use.
- `resource`: is the object family to use.
- `object`: is the object type to operate on.
- `record`: is the object instance.
- `query`: parametrized key/values for the API request.
- `payload`: body contents for the request.

All those parameters but payload are put together sequentially to build the request:

```txt
https://<salesforce-host>/services/data/<version>/<resource>/<object>/<record>?query
```

Please, refer to the [Salesforce API](https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/intro_what_is_rest_api.htm) on how to fill in values to execute requests.

#### Examples

The Salesforce target will create an account when receiving this event.

```sh
curl -v -X POST http://localhost:8080  \
    -H "content-type: application/json"  \
    -H "ce-specversion: 1.0"  \
    -H "ce-source: curl-pablo"  \
    -H "ce-type: io.triggermesh.salesforce.apicall"  \
    -H "ce-id: 123-abc" \
    -H "ce-statefulid: my-stateful-12345" \
    -H "ce-somethingelse: hello-world" \
    -H "statefulid: hello-world" \
    -d '{
          "action": "POST",
          "resource": "sobjects",
          "object": "account",
          "payload": {"Name": "test"}
        }'
```

An account can be deleted.

```sh
curl -v -X POST http://localhost:8080  \
  -H "content-type: application/json"  \
  -H "ce-specversion: 1.0"  \
  -H "ce-source: curl-pablo"  \
  -H "ce-type: my-curl-type"  \
  -H "ce-id: 123-abc" \
  -H "ce-statefulid: my-stateful-12345" \
  -H "ce-somethingelse: hello-world" \
  -H "statefulid: hello-world" \
  -d '{
        "action": "DELETE",
        "resource": "sobjects",
        "object": "account",
        "record": "0014x000005Y9SNAA0"
      }'
```

Specific fields of an account can be retrieved by using the query parameter.

```sh
curl -v -X POST http://localhost:8080  \
  -H "content-type: application/json"  \
  -H "ce-specversion: 1.0"  \
  -H "ce-source: curl-pablo"  \
  -H "ce-type: my-curl-type"  \
  -H "ce-id: 123-abc" \
  -H "ce-statefulid: my-stateful-12345" \
  -H "ce-somethingelse: hello-world" \
  -H "statefulid: hello-world" \
  -d '{
        "action": "GET",
        "resource": "sobjects",
        "object": "account",
        "record": "0014x000005VB1lAAG",
        "query": {"fields": "AccountNumber,BillingPostalCode"}
      }'
```

Salesforce uses `PATCH` to update records

```sh
curl -v -X POST http://localhost:8080  \
  -H "content-type: application/json"  \
  -H "ce-specversion: 1.0"  \
  -H "ce-source: curl-pablo"  \
  -H "ce-type: my-curl-type"  \
  -H "ce-id: 123-abc" \
  -H "ce-statefulid: my-stateful-12345" \
  -H "ce-somethingelse: hello-world" \
  -H "statefulid: hello-world" \
  -d '{
        "action": "PATCH",
        "resource": "sobjects",
        "object": "account",
        "record": "0014x000005Y9SNAA0",
        "payload": {"Name": "test2", "BillingCity" : "San Francisco"}
      }'
```
