# Hasura Target for Knative Eventing

This event target utilizes CloudEvents to send [GraphQL](https://graphql.org/)
queries to Hasura.  The Hasura Target supports two methods of invocation:

- Raw query where the payload can be passed directly to Hasura's GraphQL endpoint
- A pre-defined query where it can be invoked with the variables passed in the message

## Contents

- [Hasura Target for Knative Eventing](#hasura-target-for-knative-eventing)
  - [Contents](#contents)
  - [Prerequisites](#prerequisites)
  - [Deploying from Code](#deploying-from-code)
  - [Create Hasura Target Integration](#create-hasura-target-integration)
    - [When an Authentication Token Is Required](#when-an-authentication-token-is-required)
    - [Creating a Hasura Target](#creating-a-hasura-target)
      - [Sending a Known Query](#sending-a-known-query)
    - [Sending a Raw Query](#sending-a-raw-query)
    - [Send a Predefined Query](#send-a-predefined-query)

## Prerequisites

The Hasura base URL acting as the endpoint is a requirement.  In addition, either
an admin token or JWT token may be required to grant permission to the target
tables hosted by Hasura.

Authentication and authorization will be considered out of scope for the target guide, but
can be found in the [Hasura Guide](https://hasura.io/docs/1.0/graphql/core/auth/index.html)

## Deploying from Code

The parent config directory can be used to deploy the controller and all adapters. Please
consult the [development guide](../DEVELOPMENT.md) for information about how to deploy to
a cluster.

The adapter can be built and invoked directly.  From the top-level source directory:

```sh
make hasura-target-adapater && ./_output/hasura-target-adapter
```

Note that several environment variables will need to be set prior to invoking the adapter such as:

  - `NAMESPACE=default`           - Usually set by the Kubernetes cluster
  - `K_LOGGING_CONFIG=''`         - Define the default logging configuration
  - `K_METRICS_CONFIG='''`        - Define the Prometheus metrics configuration
  - `HASURA_ENDPOINT`             - The base URL for the target Hasura instance
  - `HASURA_JWT_TOKEN`            - An optional JWT token providing authenticated access
  - `HASURA_ADMIN_TOKEN`          - An optional admin token to provide authenticated access
  - `HASURA_DEFAULT_ROLE`         - Optional parameter used in conjunction with the JWT Token to map all queries to the defined target's role
  - `HASURA_QUERIES`              - Stringified JSON array of pre-canned queries

A full deployment example is located in the [samples](../samples/hasura) directory

## Create Hasura Target Integration

### When an Authentication Token Is Required

Any authentication token will require a secret to be created before it can be used.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: hasuratargetadminsecret
type: Opaque
stringData:
  token: <REPLACE ME WITH A REAL TOKEN>
```

### Creating a Hasura Target

The Hasura target can handle two types of events: a raw request for Hasura that
consists of a [JSON payload](https://graphql.org/learn/serving-over-http/), and
passing a premade query with the parameters in a key/value array.

Regardless of method, the Knative target will require an endpoint.

If required, the admin token or a JWT token can be specified for queries that
require authorized access.  Lastly, if a particular role is required, a `defaultRole`
can be defined.

#### Sending a Known Query

Predefined queries can be specified in the target spec using a
[sample example](../samples/hasura/200-target.yaml).  For an incoming CloudEvent
to make use of the pre-defined query, the `Ce-Subject` will need to specify the
specific query, and the `Ce-Type` must match `org.graphql.query` or
`org.graphql.query.raw`. The resulting payload must be an object of keys and
string values.


### Sending a Raw Query

```sh
curl -v http://localhost:8080 \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: org.graphql.query.raw" \
 -H "Ce-Source: awesome/instance" \
 -H "Ce-Id: aabbccdd11223344" \
 -d '{
  "query": "query MyQuery { foo { id name } }",
  "operationName": "MyQuery",
  "variables": {}
}'
```

### Send a Predefined Query
Assuming a sample target spec:

```yaml
apiVersion: targets.triggermesh.io/v1alpha1
kind: HasuraTarget
metadata:
 name: hasuratarget

spec:
  endpoint: 'http://hasura.example.com:8080' # Target Hasura instance
  queries:
    - name: MyQuery
      query: "query MyQuery($person_id: Int) { foo(where: {id: {_eq: $person_id}} ) { id name } }"

```

Then the following event could be used:
```sh
curl -v http://localhost:8080 \
 -X POST \
 -H "Content-Type: application/json" \
 -H "Ce-Specversion: 1.0" \
 -H "Ce-Type: org.graphql.query" \
 -H "Ce-Source: awesome/instance" \
 -H "Ce-Subject: MyQuery" \
 -H "Ce-Id: aabbccdd11223344" \
 -d '{"person_id": "5"}'
```
